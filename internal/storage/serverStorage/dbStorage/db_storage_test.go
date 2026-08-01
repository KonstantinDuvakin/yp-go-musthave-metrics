package dbStorage

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/migrations"
)

func newTestStorage(t *testing.T) *DBStorage {
	t.Helper()

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN не задан — пропускаю интеграционный тест")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)

	require.NoError(t, pool.Ping(context.Background()), "БД недоступна по DATABASE_DSN")

	sqlDB := stdlib.OpenDBFromPool(pool)
	require.NoError(t, migrations.RunMigrations(sqlDB))
	sqlDB.Close()

	_, err = pool.Exec(context.Background(), "TRUNCATE metrics")
	require.NoError(t, err)

	t.Cleanup(func() { pool.Close() })

	return NewDBStorage(pool)
}

func TestDBStorage_SetGauge(t *testing.T) {
	s := newTestStorage(t)

	require.NoError(t, s.SetGauge("Alloc", 42.5))

	v, ok, err := s.GetGauge("Alloc")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 42.5, v)
}

func TestDBStorage_SetGauge_Overwrites(t *testing.T) {
	s := newTestStorage(t)

	require.NoError(t, s.SetGauge("Alloc", 1.0))
	require.NoError(t, s.SetGauge("Alloc", 2.0))

	v, ok, err := s.GetGauge("Alloc")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 2.0, v, "gauge должен перезаписываться последним значением")
}

func TestDBStorage_AddCounter_Accumulates(t *testing.T) {
	s := newTestStorage(t)

	require.NoError(t, s.AddCounter("PollCount", 10))
	require.NoError(t, s.AddCounter("PollCount", 5))

	v, ok, err := s.GetCounter("PollCount")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(15), v, "counter должен накапливаться")
}

func TestDBStorage_GetGauge_NotFound(t *testing.T) {
	s := newTestStorage(t)

	v, ok, err := s.GetGauge("missing")
	require.NoError(t, err, "отсутствие метрики — не ошибка")
	require.False(t, ok)
	require.Zero(t, v)
}

func TestDBStorage_GetCounter_NotFound(t *testing.T) {
	s := newTestStorage(t)

	v, ok, err := s.GetCounter("missing")
	require.NoError(t, err)
	require.False(t, ok)
	require.Zero(t, v)
}

func TestDBStorage_GetAllGauges(t *testing.T) {
	s := newTestStorage(t)

	require.NoError(t, s.SetGauge("a", 1.5))
	require.NoError(t, s.SetGauge("b", 2.5))

	all, err := s.GetAllGauges()
	require.NoError(t, err)
	require.Equal(t, map[string]float64{"a": 1.5, "b": 2.5}, map[string]float64(all))
}

func TestDBStorage_GetAllCounters(t *testing.T) {
	s := newTestStorage(t)

	require.NoError(t, s.AddCounter("x", 3))
	require.NoError(t, s.AddCounter("y", 7))

	all, err := s.GetAllCounters()
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"x": 3, "y": 7}, map[string]int64(all))
}

func TestDBStorage_SaveMetricsBatch(t *testing.T) {
	s := newTestStorage(t)

	gv := 3.14
	var cd int64 = 7
	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &gv},
		{ID: "PollCount", MType: models.Counter, Delta: &cd},
	}
	require.NoError(t, s.SaveMetricsBatch(context.Background(), batch))

	g, ok, err := s.GetGauge("Alloc")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 3.14, g)

	c, ok, err := s.GetCounter("PollCount")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(7), c)
}

func TestDBStorage_SaveMetricsBatch_CounterAccumulates(t *testing.T) {
	s := newTestStorage(t)

	var d1, d2 int64 = 10, 5
	require.NoError(t, s.SaveMetricsBatch(context.Background(), []models.Metrics{
		{ID: "PollCount", MType: models.Counter, Delta: &d1},
		{ID: "PollCount", MType: models.Counter, Delta: &d2},
	}))

	c, ok, err := s.GetCounter("PollCount")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(15), c, "счётчики в одном батче должны накопиться (10+5)")
}

func TestDBStorage_SaveMetricsBatch_Atomic(t *testing.T) {
	s := newTestStorage(t)

	gv := 1.0
	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &gv},
		{ID: "PollCount", MType: models.Counter, Delta: nil},
	}
	require.Error(t, s.SaveMetricsBatch(context.Background(), batch))

	_, ok, err := s.GetGauge("Alloc")
	require.NoError(t, err)
	require.False(t, ok, "при ошибке в батче ранее записанный gauge должен быть откатан")
}

func TestDBStorage_SaveMetricsBatch_Empty(t *testing.T) {
	s := newTestStorage(t)
	require.NoError(t, s.SaveMetricsBatch(context.Background(), nil))
}
