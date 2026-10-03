// Package savetofile периодически сохраняет метрики в файл.
//
// Интервал сохранения и путь к файлу задаются в конфигурации сервера.
package savetofile

import (
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

// SaveToFile периодически сохраняет метрики store в файл с интервалом
// c.StoreInterval секунд, пока не будет закрыт канал cls.
//
// Предназначен для запуска в отдельной горутине. При закрытии cls выполняет
// финальное сохранение и закрывает канал done, сигнализируя о завершении.
// При c.StoreInterval <= 0 сразу закрывает done и ничего не делает.
func SaveToFile(c *config.ServerConfig, store storage.FilePersistentStorage, cls <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	if c.StoreInterval > 0 {
		ticker := time.NewTicker(time.Duration(c.StoreInterval * float64(time.Second)))
		defer ticker.Stop()

		for {
			select {
			case <-cls:
				if err := store.SaveMetricsToFile(c.StoreFile); err != nil {
					logger.Log.Warn("Failed to save metrics to file", zap.Error(err))
				}
				return
			case <-ticker.C:
				if err := store.SaveMetricsToFile(c.StoreFile); err != nil {
					logger.Log.Warn("Failed to save metrics to file", zap.Error(err))
				}
			}
		}
	}
}
