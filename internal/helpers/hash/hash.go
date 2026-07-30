package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func CreateHeaderHash(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	dst := h.Sum(nil)
	return hex.EncodeToString(dst)
}
