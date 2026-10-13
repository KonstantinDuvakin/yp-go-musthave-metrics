package updatemetrics

import (
	"context"
	"errors"
	"testing"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeStorage запоминает пакет, переданный в SaveMetricsBatch, и возвращает
// err. Остальные методы storage.MetricsStorage сервис не вызывает.
type fakeStorage struct {
	storage.MetricsStorage
	saved []models.Metrics
	err   error
}

func (f *fakeStorage) SaveMetricsBatch(_ context.Context, metrics []models.Metrics) error {
	f.saved = metrics
	return f.err
}

// fakeAudit запоминает отправленные события аудита.
type fakeAudit struct {
	events []models.Audit
}

func (f *fakeAudit) SendEvent(event models.Audit) {
	f.events = append(f.events, event)
}

func TestService_UpdateBatch(t *testing.T) {
	gv := 1.5
	dv := int64(3)
	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &gv},
		{ID: "PollCount", MType: models.Counter, Delta: &dv},
	}

	t.Run("saves batch and sends audit event", func(t *testing.T) {
		store := &fakeStorage{}
		audit := &fakeAudit{}

		err := New(store, audit).UpdateBatch(context.Background(), batch, "192.168.1.10")

		require.NoError(t, err)
		assert.Equal(t, batch, store.saved)

		require.Len(t, audit.events, 1)
		assert.Equal(t, []string{"Alloc", "PollCount"}, audit.events[0].Metrics)
		assert.Equal(t, "192.168.1.10", audit.events[0].IPAddress)
		assert.NotZero(t, audit.events[0].TS)
	})

	t.Run("save error skips audit", func(t *testing.T) {
		errDB := errors.New("db is down")
		store := &fakeStorage{err: errDB}
		audit := &fakeAudit{}

		err := New(store, audit).UpdateBatch(context.Background(), batch, "192.168.1.10")

		require.ErrorIs(t, err, errDB, "исходная ошибка хранилища должна быть доступна через errors.Is")
		assert.Empty(t, audit.events, "несохранённый пакет не должен попадать в аудит")
	})
}
