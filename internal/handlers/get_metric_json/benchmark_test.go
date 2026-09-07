package getmetricjson

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

// BenchmarkGetMetricJSON измеряет обработку запроса POST /value: разбор JSON,
// чтение метрики и кодирование ответа.
func BenchmarkGetMetricJSON(b *testing.B) {
	store := memstorage.NewMemStorage()
	_ = store.SetGauge("Alloc", 42.5)

	handler := GetMetricJSON(store)
	body := []byte(`{"id":"Alloc","type":"gauge"}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
