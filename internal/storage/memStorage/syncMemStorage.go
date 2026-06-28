package memStorage

import (
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

type SyncMemStorage struct {
	storage.ServerStorage
	path string
}

func NewSyncMemStorage(s storage.ServerStorage, path string) *SyncMemStorage {
	return &SyncMemStorage{s, path}
}

func (s *SyncMemStorage) SetGauge(name string, value float64) {
	s.ServerStorage.SetGauge(name, value)
	if err := s.SaveMetricsToFile(s.path); err != nil {
		logger.Log.Warn("Could not save metrics to file", zap.String("path", s.path), zap.Error(err))
	}
}

func (s *SyncMemStorage) AddCounter(name string, value int64) {
	s.ServerStorage.AddCounter(name, value)
	if err := s.SaveMetricsToFile(s.path); err != nil {
		logger.Log.Warn("Could not save metrics to file", zap.String("path", s.path), zap.Error(err))
	}
}
