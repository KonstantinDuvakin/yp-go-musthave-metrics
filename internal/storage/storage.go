// Package storage описывает интерфейсы хранилищ метрик и общие типы карт
// значений. Конкретные реализации (in-memory, БД, синхронная запись в файл)
// живут в подпакетах server_storage.
package storage

import (
	"context"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
)

// GaugeMap — карта имя метрики -> значение для метрик типа gauge.
type GaugeMap map[string]float64

// CounterMap — карта имя метрики -> значение для метрик типа counter.
type CounterMap map[string]int64

// MetricsStorage — хранилище метрик: чтение и запись значений gauge и
// counter, а также пакетное сохранение. Возвращаемый вторым значением
// bool у геттеров сообщает, найдена ли метрика.
type MetricsStorage interface {
	SetGauge(string, float64) error
	AddCounter(string, int64) error
	GetGauge(string) (float64, bool, error)
	GetCounter(string) (int64, bool, error)
	GetAllGauges() (GaugeMap, error)
	GetAllCounters() (CounterMap, error)
	SaveMetricsBatch(ctx context.Context, metrics []models.Metrics) error
}

// FilePersistentStorage — способность хранилища сбрасывать метрики в файл
// и восстанавливать их из файла.
type FilePersistentStorage interface {
	SaveMetricsToFile(filename string) error
	RestoreFromFile(filename string) error
}

// ServerStorage объединяет работу с метриками ([MetricsStorage]) и их
// файловую персистентность ([FilePersistentStorage]).
type ServerStorage interface {
	MetricsStorage
	FilePersistentStorage
}
