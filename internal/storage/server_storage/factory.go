// Package serverstorage выбирает и конфигурирует конкретную реализацию
// хранилища метрик сервера в зависимости от конфигурации.
package serverstorage

import (
	"context"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/save_to_file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/db_storage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/sync_mem_storage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/migrations"
	"go.uber.org/zap"
)

// NewStorage создаёт хранилище метрик согласно config.
//
// Если задан config.DB — используется PostgreSQL (с прогоном миграций),
// иначе — in-memory хранилище. Для in-memory при нулевом StoreInterval
// запись в файл выполняется синхронно, иначе — периодически в фоне; при
// включённом Restore состояние восстанавливается из файла.
//
// Возвращает хранилище, пул соединений к БД (nil для in-memory), функцию
// корректного завершения (её нужно вызвать при остановке) и ошибку.
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

		store := dbstorage.NewDBStorage(db)
		return store, db, func() { db.Close() }, nil
	}

	base := memstorage.NewMemStorage()
	var store storage.MetricsStorage = base
	var shutdown = func() {}

	if config.Restore {
		if err := base.RestoreFromFile(config.FileStoragePath); err != nil {
			logger.Log.Warn("Failed to restore data from file", zap.Error(err))
		}
	}

	if config.StoreInterval == 0 {
		syncStore := syncmemstorage.NewSyncMemStorage(base, config.FileStoragePath)
		store = syncStore
		shutdown = syncStore.Close
	} else {
		done := make(chan struct{})
		go savetofile.SaveToFile(ctx, config, base, done)
		shutdown = func() { <-done }
	}

	return store, nil, shutdown, nil
}
