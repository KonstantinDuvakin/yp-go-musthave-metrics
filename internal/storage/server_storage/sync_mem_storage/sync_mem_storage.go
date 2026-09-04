package syncmemstorage

import (
	"context"
	"fmt"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

type SyncMemStorage struct {
	store           storage.ServerStorage
	path            string
	isSavingFlag    chan struct{}
	isFinishingFlag chan struct{}
}

func NewSyncMemStorage(s storage.ServerStorage, path string) *SyncMemStorage {
	ms := &SyncMemStorage{s, path, make(chan struct{}, 1), make(chan struct{}, 1)}
	go ms.startSaving()
	return ms
}

func (s *SyncMemStorage) SetGauge(name string, value float64) error {
	err := s.store.SetGauge(name, value)
	if err != nil {
		return fmt.Errorf("error setting metric: %s : %w", name, err)
	}
	s.markSaving()
	return nil
}

func (s *SyncMemStorage) AddCounter(name string, value int64) error {
	err := s.store.AddCounter(name, value)
	if err != nil {
		return fmt.Errorf("error setting metric: %s : %w", name, err)
	}
	s.markSaving()
	return nil
}

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

func (s *SyncMemStorage) SaveMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	if err := s.store.SaveMetricsBatch(ctx, metrics); err != nil {
		return err
	}
	s.markSaving()
	return nil
}

func (s *SyncMemStorage) GetGauge(field string) (float64, bool, error) {
	return s.store.GetGauge(field)
}

func (s *SyncMemStorage) GetCounter(field string) (int64, bool, error) {
	return s.store.GetCounter(field)
}

func (s *SyncMemStorage) GetAllGauges() (storage.GaugeMap, error) {
	return s.store.GetAllGauges()
}

func (s *SyncMemStorage) GetAllCounters() (storage.CounterMap, error) {
	return s.store.GetAllCounters()
}

func (s *SyncMemStorage) SaveMetricsToFile(filename string) error {
	return s.store.SaveMetricsToFile(filename)
}

func (s *SyncMemStorage) RestoreFromFile(filename string) error {
	return s.store.RestoreFromFile(filename)
}
