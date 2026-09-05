// Package agentstorage хранит собранные агентом метрики в памяти.
// Реализация потокобезопасна и рассчитана на конкурентный доступ из
// горутин опроса и отправки.
package agentstorage

import (
	"maps"
	"sync"

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
