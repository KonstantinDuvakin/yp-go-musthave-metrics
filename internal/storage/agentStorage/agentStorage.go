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
	defer as.mu.Unlock()
	as.Gauge[name] = value
}

func (as *AgentStorage) AddCounter(name string) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.Counter[name]++
}

func (as *AgentStorage) Snapshot() (g storage.GaugeMap, c storage.CounterMap) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	g = maps.Clone(as.Gauge)
	c = maps.Clone(as.Counter)
	return g, c
}
