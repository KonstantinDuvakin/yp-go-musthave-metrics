package updatebatchmetrics

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	updatemetrics "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/update_metrics"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

// BenchmarkUpdateBatchMetrics измеряет обработку запроса POST /updates: разбор
// пакета метрик, сохранение и формирование события аудита.
func BenchmarkUpdateBatchMetrics(b *testing.B) {
	store := memstorage.NewMemStorage()
	noopAudit := auditFunc(func(models.Audit) {})
	handler := UpdateBatchMetrics(updatemetrics.New(store, noopAudit))

	body := []byte(`[
		{"id":"Alloc","type":"gauge","value":42.5},
		{"id":"HeapAlloc","type":"gauge","value":100.0},
		{"id":"PollCount","type":"counter","delta":10}
	]`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
