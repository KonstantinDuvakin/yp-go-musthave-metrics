package updatemetricjson

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

// BenchmarkUpdateMetricJSON измеряет обработку запроса POST /update: разбор
// JSON и запись метрики в хранилище.
func BenchmarkUpdateMetricJSON(b *testing.B) {
	store := memstorage.NewMemStorage()
	handler := UpdateMetricJSON(store)

	body := []byte(`{"id":"Alloc","type":"gauge","value":42.5}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
