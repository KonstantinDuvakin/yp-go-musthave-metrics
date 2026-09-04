package serverstorage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
)

func TestNewStorage_MemorySync(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	c := &config.ServerConfig{StoreInterval: 0, FileStoragePath: path, Restore: false}

	store, db, shutdown, err := NewStorage(context.Background(), c)
	require.NoError(t, err)
	require.NotNil(t, store)
	require.Nil(t, db, "без DATABASE_DSN пул БД не создаётся")
	require.NotNil(t, shutdown)

	require.NoError(t, store.SetGauge("Alloc", 1.5))
	shutdown()
}

func TestNewStorage_MemoryInterval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	c := &config.ServerConfig{StoreInterval: 3600, FileStoragePath: path, Restore: false}

	ctx, cancel := context.WithCancel(context.Background())

	store, db, shutdown, err := NewStorage(ctx, c)
	require.NoError(t, err)
	require.NotNil(t, store)
	require.Nil(t, db)
	require.NotNil(t, shutdown)

	cancel()
	shutdown()
}

func TestNewStorage_DB(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN не задан — пропускаю интеграционный тест")
	}

	c := &config.ServerConfig{DB: dsn}

	store, db, shutdown, err := NewStorage(context.Background(), c)
	require.NoError(t, err)
	require.NotNil(t, store)
	require.NotNil(t, db, "с DATABASE_DSN пул возвращается для /ping")
	require.NotNil(t, shutdown)

	require.NoError(t, db.Ping(context.Background()))
	shutdown() // db.Close()
}
