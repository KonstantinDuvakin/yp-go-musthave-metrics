package serverStorage

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует драйвер "pgx" для database/sql

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage/dbStorage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage/memStorage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage/syncMemStorage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/migrations"
	"go.uber.org/zap"
)

func NewStorage(ctx context.Context, config *config.ServerConfig) (storage.MetricsStorage, *sql.DB, func(), error) {
	if config.DB != "" {
		db, err := sql.Open("pgx", config.DB)
		if err != nil {
			logger.Log.Error("db connection error: ", zap.Error(err))
			return nil, nil, nil, err
		}

		if err = migrations.RunMigrations(db); err != nil {
			logger.Log.Error("migrations error", zap.Error(err))
			db.Close()
			return nil, nil, nil, err
		}

		store := dbStorage.NewDBStorage(db)
		return store, db, func() { db.Close() }, nil
	}

	base := memStorage.NewMemStorage()
	var store storage.MetricsStorage = base
	var shutdown = func() {}

	if config.Restore {
		if err := base.RestoreFromFile(config.FileStoragePath); err != nil {
			logger.Log.Warn("Failed to restore data from file", zap.Error(err))
		}
	}

	if config.StoreInterval == 0 {
		syncStore := syncMemStorage.NewSyncMemStorage(base, config.FileStoragePath)
		store = syncStore
		shutdown = syncStore.Close
	} else {
		done := make(chan struct{})
		go service.SaveToFile(ctx, config, base, done)
		shutdown = func() { <-done }
	}

	return store, nil, shutdown, nil
}
