package gzip

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkMiddleware измеряет путь сжатия ответа — именно он оптимизирован
// через sync.Pool. Сравнение allocs/op показывает эффект переиспользования gzip-writer'ов
func BenchmarkMiddleware(b *testing.B) {
	payload := bytes.Repeat([]byte("metrics-data "), 1024)

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
	})
	h := Middleware(final)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}
