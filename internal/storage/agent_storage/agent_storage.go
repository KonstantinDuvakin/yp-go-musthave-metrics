// Package agentstorage хранит собранные агентом метрики в памяти.
// Реализация потокобезопасна и рассчитана на конкурентный доступ из
// горутин опроса и отправки.
package agentstorage

import (
	"maps"
	"sync"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

// AgentStorage — потокобезопасное in-memory хранилище метрик агента.
type AgentStorage struct {
	Gauge   storage.GaugeMap
	Counter storage.CounterMap
	mu      sync.RWMutex
}

// NewAgentStorage создаёт пустое [AgentStorage] с проинициализированными
// картами метрик.
func NewAgentStorage() *AgentStorage {
	return &AgentStorage{
		Gauge:   make(storage.GaugeMap),
		Counter: make(storage.CounterMap),
		mu:      sync.RWMutex{},
	}
}

// SetGauge устанавливает значение gauge-метрики name, перезаписывая прежнее.
func (as *AgentStorage) SetGauge(name string, value float64) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.Gauge[name] = value
}

// AddCounter увеличивает counter-метрику name на единицу.
func (as *AgentStorage) AddCounter(name string) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.Counter[name]++
}

// Snapshot возвращает независимые копии мап gauge- и counter-метрик,
// безопасные для чтения без блокировки.
func (as *AgentStorage) Snapshot() (g storage.GaugeMap, c storage.CounterMap) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	g = maps.Clone(as.Gauge)
	c = maps.Clone(as.Counter)
	return g, c
}

// CollectMetrics собирает текущие метрики хранилища в пакет для отправки
// на сервер. Gauge-метрики передаются в поле Value, counter-метрики — в
// поле Delta. Данные берутся из [AgentStorage.Snapshot], поэтому пакет
// не зависит от последующих изменений хранилища. Порядок метрик в пакете
// не определён. Для пустого хранилища возвращает nil.
func (as *AgentStorage) CollectMetrics() []models.Metrics {
	gauges, counters := as.Snapshot()
	metricsBatch := make([]models.Metrics, 0, len(gauges)+len(counters))

	for name, value := range gauges {
		gaugeMetric := models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		}

		metricsBatch = append(metricsBatch, gaugeMetric)
	}

	for name, value := range counters {
		counterMetric := models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &value,
		}

		metricsBatch = append(metricsBatch, counterMetric)
	}

	if len(metricsBatch) == 0 {
		return nil
	}

	return metricsBatch
}
