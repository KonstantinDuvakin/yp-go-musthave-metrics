package mem_storage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"sync"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

type MemStorage struct {
	Gauge   storage.GaugeMap
	Counter storage.CounterMap
	mu      sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		Gauge:   make(storage.GaugeMap),
		Counter: make(storage.CounterMap),
		mu:      sync.RWMutex{},
	}
}

func (ms *MemStorage) SetGauge(field string, value float64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.Gauge[field] = value
	return nil
}

func (ms *MemStorage) AddCounter(field string, value int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.Counter[field] += value
	return nil
}

func (ms *MemStorage) GetGauge(field string) (float64, bool, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	res, ok := ms.Gauge[field]
	return res, ok, nil
}

func (ms *MemStorage) GetCounter(field string) (int64, bool, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	res, ok := ms.Counter[field]
	return res, ok, nil
}

func (ms *MemStorage) GetAllGauges() (storage.GaugeMap, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return maps.Clone(ms.Gauge), nil
}

func (ms *MemStorage) GetAllCounters() (storage.CounterMap, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return maps.Clone(ms.Counter), nil
}

func (ms *MemStorage) SaveMetricsToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	enc := json.NewEncoder(writer)

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for id, delta := range ms.Counter {
		metric := models.Metrics{ID: id, MType: models.Counter, Delta: &delta}
		if err = enc.Encode(metric); err != nil {
			return err
		}
	}

	for id, value := range ms.Gauge {
		metric := models.Metrics{ID: id, MType: models.Gauge, Value: &value}
		if err = enc.Encode(metric); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func (ms *MemStorage) RestoreFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		data := scanner.Bytes()

		metric := models.Metrics{}
		if err = json.Unmarshal(data, &metric); err != nil {
			return err
		}

		switch metric.MType {
		case models.Counter:
			_ = ms.AddCounter(metric.ID, *metric.Delta)
		case models.Gauge:
			_ = ms.SetGauge(metric.ID, *metric.Value)
		default:
			return errors.New("invalid metric type")
		}
	}

	return scanner.Err()
}

func (ms *MemStorage) SaveMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				return fmt.Errorf("gauge %s: value is nil", m.ID)
			}
			_ = ms.SetGauge(m.ID, *m.Value)
		case models.Counter:
			if m.Delta == nil {
				return fmt.Errorf("counter %s: delta is nil", m.ID)
			}
			_ = ms.AddCounter(m.ID, *m.Delta)
		default:
			return fmt.Errorf("unknown metric type: %s", m.MType)
		}
	}
	return nil
}
