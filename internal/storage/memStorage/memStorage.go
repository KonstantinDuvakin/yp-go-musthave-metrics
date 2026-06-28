package memStorage

import (
	"bufio"
	"encoding/json"
	"errors"
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

func (ms *MemStorage) SetGauge(field string, value float64) {
	ms.mu.Lock()
	ms.Gauge[field] = value
	ms.mu.Unlock()
}

func (ms *MemStorage) AddCounter(field string, value int64) {
	ms.mu.Lock()
	ms.Counter[field] += value
	ms.mu.Unlock()
}

func (ms *MemStorage) GetGauge(field string) (float64, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	res, ok := ms.Gauge[field]
	return res, ok
}

func (ms *MemStorage) GetCounter(field string) (int64, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	res, ok := ms.Counter[field]
	return res, ok
}

func (ms *MemStorage) GetAllGauges() storage.GaugeMap {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return maps.Clone(ms.Gauge)
}

func (ms *MemStorage) GetAllCounters() storage.CounterMap {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return maps.Clone(ms.Counter)
}

func (ms *MemStorage) SaveMetricsToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	dec := json.NewEncoder(writer)

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for id, delta := range ms.Counter {
		metric := models.Metrics{ID: id, MType: models.Counter, Delta: &delta}
		if err = dec.Encode(metric); err != nil {
			return err
		}
	}

	for id, value := range ms.Gauge {
		metric := models.Metrics{ID: id, MType: models.Gauge, Value: &value}
		if err = dec.Encode(metric); err != nil {
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
			ms.AddCounter(metric.ID, *metric.Delta)
		case models.Gauge:
			ms.SetGauge(metric.ID, *metric.Value)
		default:
			return errors.New("invalid metric type")
		}
	}

	return scanner.Err()
}
