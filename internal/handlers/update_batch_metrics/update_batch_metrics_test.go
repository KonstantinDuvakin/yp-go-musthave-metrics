package update_batch_metrics

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

func doRequest(t *testing.T, store *mem_storage.MemStorage, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	UpdateBatchMetrics(store, func(models.Audit) {})(rec, req)
	return rec
}

func TestUpdateBatchMetrics_OK(t *testing.T) {
	store := mem_storage.NewMemStorage()

	gv := 2.5
	var cd int64 = 4
	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &gv},
		{ID: "PollCount", MType: models.Counter, Delta: &cd},
	}
	body, err := json.Marshal(batch)
	require.NoError(t, err)

	rec := doRequest(t, store, body)
	require.Equal(t, http.StatusOK, rec.Code)

	g, ok, _ := store.GetGauge("Alloc")
	require.True(t, ok)
	require.Equal(t, 2.5, g)

	c, ok, _ := store.GetCounter("PollCount")
	require.True(t, ok)
	require.Equal(t, int64(4), c)
}

func TestUpdateBatchMetrics_InvalidJSON(t *testing.T) {
	store := mem_storage.NewMemStorage()
	rec := doRequest(t, store, []byte(`{ broken`))
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// counter без delta → SaveMetricsBatch вернёт ошибку → 500
func TestUpdateBatchMetrics_StorageError(t *testing.T) {
	store := mem_storage.NewMemStorage()
	rec := doRequest(t, store, []byte(`[{"id":"PollCount","type":"counter"}]`))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
