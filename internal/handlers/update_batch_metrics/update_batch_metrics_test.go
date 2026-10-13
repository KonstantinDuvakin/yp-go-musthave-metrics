package updatebatchmetrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	updatemetrics "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/update_metrics"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

// auditFunc позволяет передать обычную функцию туда, где ждут
// updatemetrics.AuditSender.
type auditFunc func(models.Audit)

func (f auditFunc) SendEvent(event models.Audit) { f(event) }

// newHandler собирает обработчик поверх настоящего сервиса с хранилищем store.
func newHandler(store *memstorage.MemStorage, audit auditFunc) http.HandlerFunc {
	return UpdateBatchMetrics(updatemetrics.New(store, audit))
}

func doRequest(t *testing.T, handler http.HandlerFunc, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestUpdateBatchMetrics_OK(t *testing.T) {
	store := memstorage.NewMemStorage()
	var events []models.Audit
	handler := newHandler(store, func(e models.Audit) { events = append(events, e) })

	gv := 2.5
	var cd int64 = 4
	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &gv},
		{ID: "PollCount", MType: models.Counter, Delta: &cd},
	}
	body, err := json.Marshal(batch)
	require.NoError(t, err)

	rec := doRequest(t, handler, body)
	require.Equal(t, http.StatusOK, rec.Code)

	g, ok, _ := store.GetGauge("Alloc")
	require.True(t, ok)
	require.Equal(t, 2.5, g)

	c, ok, _ := store.GetCounter("PollCount")
	require.True(t, ok)
	require.Equal(t, int64(4), c)

	// Аудит пишет сервис; обработчик передаёт ему адрес отправителя.
	require.Len(t, events, 1)
	require.Equal(t, []string{"Alloc", "PollCount"}, events[0].Metrics)
	// Заголовка X-Real-IP нет — берётся адрес соединения без порта
	// (httptest.NewRequest задаёт RemoteAddr 192.0.2.1:1234).
	require.Equal(t, "192.0.2.1", events[0].IPAddress)
}

func TestUpdateBatchMetrics_AuditIP(t *testing.T) {
	tests := []struct {
		name    string
		realIP  string // значение X-Real-IP; "" — заголовка нет
		wantIP  string
		comment string
	}{
		{name: "x-real-ip header", realIP: "10.0.0.5", wantIP: "10.0.0.5", comment: "IP, который прислал агент"},
		{name: "x-real-ip is normalized", realIP: "2001:DB8:0:0::1", wantIP: "2001:db8::1", comment: "та же запись, что даёт gRPC-версия clientIP"},
		{name: "invalid x-real-ip", realIP: "not-an-ip", wantIP: "192.0.2.1", comment: "некорректный заголовок — адрес соединения без порта"},
		{name: "no header", realIP: "", wantIP: "192.0.2.1", comment: "адрес соединения без порта"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var events []models.Audit
			handler := newHandler(memstorage.NewMemStorage(), func(e models.Audit) { events = append(events, e) })

			req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader([]byte(`[{"id":"Alloc","type":"gauge","value":1}]`)))
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}
			rec := httptest.NewRecorder()
			handler(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			require.Len(t, events, 1)
			require.Equal(t, tt.wantIP, events[0].IPAddress, tt.comment)
		})
	}
}

func TestUpdateBatchMetrics_InvalidJSON(t *testing.T) {
	updater := &stubUpdater{}
	rec := doRequest(t, UpdateBatchMetrics(updater), []byte(`{ broken`))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Zero(t, updater.calls, "некорректный JSON не должен доходить до сервиса")
}

// counter без delta → SaveMetricsBatch вернёт ошибку → 500
func TestUpdateBatchMetrics_StorageError(t *testing.T) {
	store := memstorage.NewMemStorage()
	rec := doRequest(t, newHandler(store, func(models.Audit) {}), []byte(`[{"id":"PollCount","type":"counter"}]`))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

// stubUpdater — заглушка batchUpdater для проверки самого обработчика без сервиса.
type stubUpdater struct {
	calls int
	err   error
}

func (s *stubUpdater) UpdateBatch(context.Context, []models.Metrics, string) error {
	s.calls++
	return s.err
}

func TestUpdateBatchMetrics_UpdaterError(t *testing.T) {
	updater := &stubUpdater{err: errors.New("db is down")}
	rec := doRequest(t, UpdateBatchMetrics(updater), []byte(`[{"id":"Alloc","type":"gauge","value":1}]`))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, 1, updater.calls)
}

func TestClientIP_RemoteAddrWithoutPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	req.RemoteAddr = "pipe" // адрес без порта, например у unix-сокета

	require.Equal(t, "pipe", clientIP(req), "адрес без порта берётся как есть")
}
