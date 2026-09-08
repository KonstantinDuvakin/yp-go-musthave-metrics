// Package hash предоставляет HTTP-middleware, проверяющее подпись входящего
// запроса и подписывающее ответ по алгоритму HMAC-SHA256.
package hash

import (
	"bytes"
	"crypto/hmac"
	"io"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/hash"
)

type hashResponseWriter struct {
	http.ResponseWriter
	responseDataBuffer bytes.Buffer
	statusCode         int
}

func (hrw *hashResponseWriter) WriteHeader(code int) {
	hrw.statusCode = code
}

func (hrw *hashResponseWriter) Write(b []byte) (int, error) {
	return hrw.responseDataBuffer.Write(b)
}

// HashMiddleware возвращает middleware, проверяющее и добавляющее подпись
// HMAC-SHA256 с ключом key.
//
// При пустом key middleware отключено и пропускает запросы без изменений.
// Иначе: если в запросе есть заголовок HashSHA256, тело проверяется на
// соответствие подписи (при несовпадении — 400); ответ подписывается тем
// же ключом и подпись добавляется в заголовок HashSHA256.
func HashMiddleware(key string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if key == "" {
				h.ServeHTTP(rw, r)
				return
			}

			hrw := &hashResponseWriter{ResponseWriter: rw, statusCode: http.StatusOK}

			gotHash := r.Header.Get("HashSHA256")

			if gotHash != "" {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(rw, "cannot read body", http.StatusBadRequest)
					return
				}
				r.Body.Close()

				bodyHash := hash.CreateHeaderHash(body, key)

				if !hmac.Equal([]byte(bodyHash), []byte(gotHash)) {
					rw.WriteHeader(http.StatusBadRequest)
					return
				}

				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			h.ServeHTTP(hrw, r)

			respBody := hrw.responseDataBuffer.Bytes()
			if key != "" {
				rw.Header().Add("HashSHA256", hash.CreateHeaderHash(respBody, key))
			}
			rw.WriteHeader(hrw.statusCode)
			rw.Write(respBody)
		})
	}
}
