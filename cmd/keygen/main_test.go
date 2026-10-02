package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_GeneratesUsableKeyPair(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, run(dir, minBits, false))

	privateKey, err := crypto.ReadPrivateKey(filepath.Join(dir, privateKeyFile))
	require.NoError(t, err, "приватный ключ должен читаться helpers/crypto")

	publicKey, err := crypto.ReadPublicKey(filepath.Join(dir, publicKeyFile))
	require.NoError(t, err, "публичный ключ должен читаться helpers/crypto")

	assert.True(t, publicKey.Equal(&privateKey.PublicKey), "публичный ключ должен соответствовать приватному")
	assert.Equal(t, minBits, privateKey.N.BitLen())

	message := []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)
	encrypted, err := crypto.EncryptBodyWithKey(message, publicKey)
	require.NoError(t, err)

	decrypted, err := crypto.DecryptBodyWithKey(encrypted, privateKey)
	require.NoError(t, err)
	assert.Equal(t, message, decrypted)
}

func TestRun_CreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "keys")

	require.NoError(t, run(dir, minBits, false))

	assert.FileExists(t, filepath.Join(dir, privateKeyFile))
	assert.FileExists(t, filepath.Join(dir, publicKeyFile))
}

func TestRun_RejectsSmallKey(t *testing.T) {
	dir := t.TempDir()

	require.Error(t, run(dir, 1024, false))

	assert.NoFileExists(t, filepath.Join(dir, privateKeyFile))
	assert.NoFileExists(t, filepath.Join(dir, publicKeyFile))
}

func TestRun_ExistingFiles(t *testing.T) {
	for _, existing := range []string{privateKeyFile, publicKeyFile} {
		t.Run(existing+" exists, no force", func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, existing)
			require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

			require.Error(t, run(dir, minBits, false))

			got, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, "old", string(got), "существующий файл не должен меняться")

			for _, name := range []string{privateKeyFile, publicKeyFile} {
				if name != existing {
					assert.NoFileExists(t, filepath.Join(dir, name), "второй файл не должен создаваться")
				}
			}
		})
	}

	t.Run("force overwrites", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, run(dir, minBits, false))

		before, err := os.ReadFile(filepath.Join(dir, privateKeyFile))
		require.NoError(t, err)

		require.NoError(t, run(dir, minBits, true))

		after, err := os.ReadFile(filepath.Join(dir, privateKeyFile))
		require.NoError(t, err)
		assert.NotEqual(t, before, after, "с -force ключ должен быть перегенерирован")
	})
}
