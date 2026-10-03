package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeKeyPair генерирует RSA-пару и сохраняет её в dir в формате PEM (PKCS#1).
func writeKeyPair(t *testing.T, dir string) (pubPath, privPath string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privPath = filepath.Join(dir, "private.pem")
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	require.NoError(t, os.WriteFile(privPath, privPEM, 0o600))

	pubPath = filepath.Join(dir, "public.pem")
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	})
	require.NoError(t, os.WriteFile(pubPath, pubPEM, 0o644))

	return pubPath, privPath
}

// readKeyPair генерирует пару и читает её через тестируемые функции.
func readKeyPair(t *testing.T) (*rsa.PublicKey, *rsa.PrivateKey) {
	t.Helper()

	pubPath, privPath := writeKeyPair(t, t.TempDir())

	pub, err := ReadPublicKey(pubPath)
	require.NoError(t, err)

	priv, err := ReadPrivateKey(privPath)
	require.NoError(t, err)

	return pub, priv
}

func TestReadKeys(t *testing.T) {
	dir := t.TempDir()
	pubPath, privPath := writeKeyPair(t, dir)

	garbage := filepath.Join(dir, "garbage.pem")
	require.NoError(t, os.WriteFile(garbage, []byte("not a pem"), 0o600))

	t.Run("valid", func(t *testing.T) {
		pub, err := ReadPublicKey(pubPath)
		require.NoError(t, err)
		assert.NotNil(t, pub)

		priv, err := ReadPrivateKey(privPath)
		require.NoError(t, err)
		assert.NotNil(t, priv)

		assert.True(t, pub.Equal(&priv.PublicKey), "ключи должны быть парой")
	})

	// Ни в одном случае функции не должны вернуть (nil, nil).
	errCases := []struct {
		name string
		path string
	}{
		{name: "wrong extension", path: filepath.Join(dir, "key.txt")},
		{name: "missing file", path: filepath.Join(dir, "missing.pem")},
		{name: "no PEM block", path: garbage},
	}
	for _, tc := range errCases {
		t.Run("public/"+tc.name, func(t *testing.T) {
			key, err := ReadPublicKey(tc.path)
			assert.Error(t, err)
			assert.Nil(t, key)
		})
		t.Run("private/"+tc.name, func(t *testing.T) {
			key, err := ReadPrivateKey(tc.path)
			assert.Error(t, err)
			assert.Nil(t, key)
		})
	}

	t.Run("private key as public", func(t *testing.T) {
		key, err := ReadPublicKey(privPath)
		assert.Error(t, err)
		assert.Nil(t, key)
	})

	t.Run("public key as private", func(t *testing.T) {
		key, err := ReadPrivateKey(pubPath)
		assert.Error(t, err)
		assert.Nil(t, key)
	})
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	pub, priv := readKeyPair(t)

	tests := []struct {
		name string
		msg  []byte
	}{
		{name: "empty", msg: []byte{}},
		{name: "small json", msg: []byte(`{"id":"Alloc","type":"gauge","value":1.5}`)},
		// Больше предела RSA-OAEP (190 байт) — ради этого и нужна гибридная схема.
		{name: "larger than RSA limit", msg: bytes.Repeat([]byte("a"), 64<<10)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := EncryptBodyWithKey(tc.msg, pub)
			require.NoError(t, err)

			// [RSA(aesKey)][nonce 12][шифротекст][тег 16]
			assert.Len(t, encrypted, priv.Size()+12+len(tc.msg)+16)

			decrypted, err := DecryptBodyWithKey(encrypted, priv)
			require.NoError(t, err)
			// Для пустого сообщения GCM возвращает nil, а не []byte{}.
			assert.Equal(t, string(tc.msg), string(decrypted))
		})
	}
}

// Каждый вызов использует новые AES-ключ и nonce.
func TestEncrypt_NonDeterministic(t *testing.T) {
	pub, _ := readKeyPair(t)
	msg := []byte("same message")

	a, err := EncryptBodyWithKey(msg, pub)
	require.NoError(t, err)
	b, err := EncryptBodyWithKey(msg, pub)
	require.NoError(t, err)

	assert.NotEqual(t, a, b)
}

func TestDecrypt_Errors(t *testing.T) {
	pub, priv := readKeyPair(t)
	_, otherPriv := readKeyPair(t)

	encrypted, err := EncryptBodyWithKey([]byte("secret"), pub)
	require.NoError(t, err)

	flip := func(i int) []byte {
		b := bytes.Clone(encrypted)
		b[i] ^= 0x01
		return b
	}

	tests := []struct {
		name string
		body []byte
		key  *rsa.PrivateKey
	}{
		{name: "empty body", body: nil, key: priv},
		{name: "shorter than RSA block", body: encrypted[:priv.Size()-1], key: priv},
		{name: "no room for nonce and tag", body: encrypted[:priv.Size()+12+15], key: priv},
		{name: "foreign private key", body: encrypted, key: otherPriv},
		{name: "tampered encrypted key", body: flip(0), key: priv},
		{name: "tampered nonce", body: flip(priv.Size()), key: priv},
		{name: "tampered tag", body: flip(len(encrypted) - 1), key: priv},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				decrypted, err := DecryptBodyWithKey(tc.body, tc.key)
				assert.Error(t, err)
				assert.Nil(t, decrypted)
			})
		})
	}
}
