// Package crypto предоставляет HTTP-middleware, расшифровывающее тело
// запроса, которое агент зашифровал публичным RSA-ключом (см.
// [crypto.EncryptBodyWithKey]).
//
// Middleware должно стоять в цепочке раньше gzip и проверки подписи:
// агент сжимает и подписывает тело до шифрования, поэтому сервер сначала
// расшифровывает, затем распаковывает и только потом проверяет подпись.
package crypto

import (
	"bytes"
	"crypto/rsa"
	"errors"
	"io"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/crypto"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"go.uber.org/zap"
)

// Middleware возвращает middleware, расшифровывающее тело запроса ключом
// privateKey.
//
// Расшифровываются только запросы с заголовком X-Encrypted: true; остальные
// пропускаются без изменений. Для зашифрованного запроса middleware отвечает:
//   - 400, если privateKey равен nil (сервер запущен без ключа);
//   - 413, если тело больше 1 МБ;
//   - 400, если тело не удалось прочитать или расшифровать.
//
// При успехе тело запроса заменяется расшифрованными данными.
func Middleware(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			encryptHeader := r.Header.Get("X-Encrypted")

			shouldDecrypt := encryptHeader == "true"

			if privateKey == nil && shouldDecrypt {
				logger.Log.Error("Private key not provided")
				http.Error(rw, "Encryption not supported", http.StatusBadRequest)
				return
			}

			if shouldDecrypt {
				body, err := io.ReadAll(http.MaxBytesReader(rw, r.Body, 1<<20))
				if err != nil {
					if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
						logger.Log.Error("Body too large", zap.Error(err))
						http.Error(rw, "Body too large", http.StatusRequestEntityTooLarge)
						return
					}
					logger.Log.Error("Error read body", zap.Error(err))
					http.Error(rw, "Can't read body", http.StatusBadRequest)
					return
				}

				decryptBody, err := crypto.DecryptBodyWithKey(body, privateKey)
				if err != nil {
					logger.Log.Error("Error decrypt request body", zap.Error(err))
					http.Error(rw, "Can't read body", http.StatusBadRequest)
					return
				}

				r.Body = io.NopCloser(bytes.NewReader(decryptBody))
			}

			next.ServeHTTP(rw, r)
		})
	}
}
