package storage

import (
	"context"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
)

type GaugeMap map[string]float64
type CounterMap map[string]int64

type MetricsStorage interface {
	SetGauge(string, float64) error
	AddCounter(string, int64) error
	GetGauge(string) (float64, bool, error)
	GetCounter(string) (int64, bool, error)
	GetAllGauges() (GaugeMap, error)
	GetAllCounters() (CounterMap, error)
	SaveMetricsBatch(ctx context.Context, metrics []models.Metrics) error
}

type FilePersistentStorage interface {
	SaveMetricsToFile(filename string) error
	RestoreFromFile(filename string) error
}

type ServerStorage interface {
	MetricsStorage
	FilePersistentStorage
}
