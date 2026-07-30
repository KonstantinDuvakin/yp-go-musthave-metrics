package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

// независимый эталон: HMAC-SHA256(data, key) в hex
func referenceHash(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func TestCreateHeaderHash_MatchesHMAC(t *testing.T) {
	data := []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)
	const key = "secret"

	require.Equal(t, referenceHash(data, key), CreateHeaderHash(data, key),
		"должен быть HMAC-SHA256 от тела с ключом, в hex")
}

func TestCreateHeaderHash_Deterministic(t *testing.T) {
	data := []byte("body")
	require.Equal(t, CreateHeaderHash(data, "k"), CreateHeaderHash(data, "k"))
}

func TestCreateHeaderHash_KeySensitive(t *testing.T) {
	data := []byte("body")
	require.NotEqual(t, CreateHeaderHash(data, "k1"), CreateHeaderHash(data, "k2"),
		"разные ключи → разный хэш")
}

func TestCreateHeaderHash_DataSensitive(t *testing.T) {
	require.NotEqual(t, CreateHeaderHash([]byte("a"), "k"), CreateHeaderHash([]byte("b"), "k"),
		"разные данные → разный хэш")
}

func TestCreateHeaderHash_HexLength(t *testing.T) {
	// SHA256 = 32 байта → 64 hex-символа
	require.Len(t, CreateHeaderHash([]byte("x"), "k"), 64)
}
