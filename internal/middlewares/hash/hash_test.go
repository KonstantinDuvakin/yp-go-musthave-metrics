package hash

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	helper "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/hash"
)

const testKey = "secret"

// хендлер, который вычитывает тело запроса и пишет фиксированный ответ
func okHandler(respBody string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respBody))
	})
}

// Без ключа middleware ничего не проверяет и не подписывает.
func TestHashMiddleware_NoKey(t *testing.T) {
	h := HashMiddleware("")(okHandler("pong"))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("data"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Header().Get("HashSHA256"), "без ключа ответ не подписывается")
	require.Equal(t, "pong", rec.Body.String())
}

// Верный хэш запроса → 200, а ответ подписан хэшем от тела ответа.
func TestHashMiddleware_ValidHash(t *testing.T) {
	reqBody := []byte("request-data")
	h := HashMiddleware(testKey)(okHandler("pong"))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqBody))
	req.Header.Set("HashSHA256", helper.CreateHeaderHash(reqBody, testKey))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "pong", rec.Body.String())
	require.Equal(t, helper.CreateHeaderHash([]byte("pong"), testKey), rec.Header().Get("HashSHA256"),
		"ответ должен быть подписан хэшем от тела ответа")
}

// Неверный хэш запроса → 400, хендлер не вызывается.
func TestHashMiddleware_InvalidHash(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	h := HashMiddleware(testKey)(next)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("request-data"))
	req.Header.Set("HashSHA256", "deadbeef") // заведомо неверный
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.False(t, called, "при несовпадении хэша хендлер не должен вызываться")
}

// Ключ есть, но клиент хэш не прислал → пропускаем (лениво), ответ подписываем.
func TestHashMiddleware_NoHashHeaderLenient(t *testing.T) {
	h := HashMiddleware(testKey)(okHandler("pong"))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("data"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, helper.CreateHeaderHash([]byte("pong"), testKey), rec.Header().Get("HashSHA256"))
}

// Краевой случай: хендлер пишет тело, но НЕ вызывает WriteHeader явно.
// Тогда неявно должен быть статус 200.
func TestHashMiddleware_HandlerWithoutWriteHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pong")) // без WriteHeader
	})
	h := HashMiddleware(testKey)(next)

	reqBody := []byte("data")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqBody))
	req.Header.Set("HashSHA256", helper.CreateHeaderHash(reqBody, testKey))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "без явного WriteHeader статус ответа должен быть 200")
	require.Equal(t, "pong", rec.Body.String())
}
