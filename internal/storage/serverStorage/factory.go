package serverStorage

import (
	"context"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/saveToFile"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage/dbStorage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage/memStorage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage/syncMemStorage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/migrations"
	"go.uber.org/zap"
)

func NewStorage(ctx context.Context, config *config.ServerConfig) (storage.MetricsStorage, *pgxpool.Pool, func(), error) {
	if config.DB != "" {
		db, err := pgxpool.New(ctx, config.DB)
		if err != nil {
			logger.Log.Error("db connection error: ", zap.Error(err))
			return nil, nil, nil, err
		}
		sqlDB := stdlib.OpenDBFromPool(db)
		defer sqlDB.Close()

		if err = migrations.RunMigrations(sqlDB); err != nil {
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
		go saveToFile.SaveToFile(ctx, config, base, done)
		shutdown = func() { <-done }
	}

	return store, nil, shutdown, nil
}
