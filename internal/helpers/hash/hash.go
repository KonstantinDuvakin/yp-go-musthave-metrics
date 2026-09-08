// Package hash вычисляет HMAC-SHA256 подпись для проверки целостности
// передаваемых данных.
package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// CreateHeaderHash возвращает HMAC-SHA256 подпись data ключом key в виде
// шестнадцатеричной строки. Значение помещается в заголовок HashSHA256.
func CreateHeaderHash(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	dst := h.Sum(nil)
	return hex.EncodeToString(dst)
}
