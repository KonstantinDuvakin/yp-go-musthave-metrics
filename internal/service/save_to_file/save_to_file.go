package save_to_file

import (
	"context"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

func SaveToFile(ctx context.Context, c *config.ServerConfig, store storage.FilePersistentStorage, done chan<- struct{}) {
	defer close(done)

	if c.StoreInterval > 0 {
		ticker := time.NewTicker(time.Duration(c.StoreInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				if err := store.SaveMetricsToFile(c.FileStoragePath); err != nil {
					logger.Log.Warn("Failed to save metrics to file", zap.Error(err))
				}
				return
			case <-ticker.C:
				if err := store.SaveMetricsToFile(c.FileStoragePath); err != nil {
					logger.Log.Warn("Failed to save metrics to file", zap.Error(err))
				}
			}
		}
	}
}
