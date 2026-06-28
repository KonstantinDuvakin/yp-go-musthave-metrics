package agentStorage

import (
	"maps"
	"sync"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

type AgentStorage struct {
	Gauge   storage.GaugeMap
	Counter storage.CounterMap
	mu      sync.RWMutex
}

func NewAgentStorage() *AgentStorage {
	return &AgentStorage{
		Gauge:   make(storage.GaugeMap),
		Counter: make(storage.CounterMap),
		mu:      sync.RWMutex{},
	}
}

func (as *AgentStorage) SetGauge(name string, value float64) {
	as.mu.Lock()
	as.Gauge[name] = value
	defer as.mu.Unlock()
}

func (as *AgentStorage) AddCounter(name string) {
	as.mu.Lock()
	as.Counter[name]++
	defer as.mu.Unlock()
}

func (as *AgentStorage) Snapshot() (g storage.GaugeMap, c storage.CounterMap) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	g = maps.Clone(as.Gauge)
	c = maps.Clone(as.Counter)
	return g, c
}
