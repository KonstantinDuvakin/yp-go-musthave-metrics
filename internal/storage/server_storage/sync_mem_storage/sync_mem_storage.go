// Package syncmemstorage оборачивает серверное хранилище и синхронно
// сбрасывает состояние в файл после каждой изменяющей операции. Используется,
// когда интервал сохранения на диск равен нулю.
package syncmemstorage

import (
	"context"
	"fmt"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

// SyncMemStorage декорирует [storage.ServerStorage], инициируя запись в файл
// после каждого изменения метрик. Запись выполняется в отдельной горутине,
// чтобы не блокировать обработку запросов.
type SyncMemStorage struct {
	store           storage.ServerStorage
	path            string
	isSavingFlag    chan struct{}
	isFinishingFlag chan struct{}
}

// NewSyncMemStorage оборачивает s синхронным сохранением в файл path и
// запускает фоновую горутину записи. Останавливать её нужно через
// [SyncMemStorage.Close].
func NewSyncMemStorage(s storage.ServerStorage, path string) *SyncMemStorage {
	ms := &SyncMemStorage{s, path, make(chan struct{}, 1), make(chan struct{}, 1)}
	go ms.startSaving()
	return ms
}

// SetGauge устанавливает gauge-метрику и инициирует сохранение в файл.
func (s *SyncMemStorage) SetGauge(name string, value float64) error {
	err := s.store.SetGauge(name, value)
	if err != nil {
		return fmt.Errorf("error setting metric: %s : %w", name, err)
	}
	s.markSaving()
	return nil
}

// AddCounter увеличивает counter-метрику и инициирует сохранение в файл.
func (s *SyncMemStorage) AddCounter(name string, value int64) error {
	err := s.store.AddCounter(name, value)
	if err != nil {
		return fmt.Errorf("error setting metric: %s : %w", name, err)
	}
	s.markSaving()
	return nil
}

// Close останавливает фоновую запись, выполняет финальное сохранение в файл
// и дожидается завершения горутины. После вызова хранилище использовать
// нельзя.
func (s *SyncMemStorage) Close() {
	close(s.isSavingFlag)
	<-s.isFinishingFlag
}

func (s *SyncMemStorage) markSaving() {
	select {
	case s.isSavingFlag <- struct{}{}:
	default:
	}
}

func (s *SyncMemStorage) startSaving() {
	for range s.isSavingFlag {
		if err := s.store.SaveMetricsToFile(s.path); err != nil {
			logger.Log.Warn("Could not save metrics to file", zap.String("path", s.path), zap.Error(err))
		}
	}

	if err := s.store.SaveMetricsToFile(s.path); err != nil {
		logger.Log.Warn("Could not save metrics to file", zap.String("path", s.path), zap.Error(err))
	}
	close(s.isFinishingFlag)
}

// SaveMetricsBatch сохраняет пакет метрик и инициирует запись в файл.
func (s *SyncMemStorage) SaveMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	if err := s.store.SaveMetricsBatch(ctx, metrics); err != nil {
		return err
	}
	s.markSaving()
	return nil
}

// GetGauge возвращает значение gauge-метрики field и признак её наличия.
func (s *SyncMemStorage) GetGauge(field string) (float64, bool, error) {
	return s.store.GetGauge(field)
}

// GetCounter возвращает значение counter-метрики field и признак её наличия.
func (s *SyncMemStorage) GetCounter(field string) (int64, bool, error) {
	return s.store.GetCounter(field)
}

// GetAllGauges возвращает копию карты всех gauge-метрик.
func (s *SyncMemStorage) GetAllGauges() (storage.GaugeMap, error) {
	return s.store.GetAllGauges()
}

// GetAllCounters возвращает копию карты всех counter-метрик.
func (s *SyncMemStorage) GetAllCounters() (storage.CounterMap, error) {
	return s.store.GetAllCounters()
}

// SaveMetricsToFile сохраняет метрики в файл filename.
func (s *SyncMemStorage) SaveMetricsToFile(filename string) error {
	return s.store.SaveMetricsToFile(filename)
}

// RestoreFromFile восстанавливает метрики из файла filename.
func (s *SyncMemStorage) RestoreFromFile(filename string) error {
	return s.store.RestoreFromFile(filename)
}
