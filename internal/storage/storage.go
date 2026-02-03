package storage

import (
	"maps"
	"sync"
)

type GaugeMap map[string]float64
type CounterMap map[string]int64

type MemStorage struct {
	Gauge   GaugeMap
	Counter CounterMap
	mu      *sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		Gauge:   make(GaugeMap),
		Counter: make(CounterMap),
		mu:      &sync.RWMutex{},
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

func (ms *MemStorage) GetAllGauges() GaugeMap {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return maps.Clone(ms.Gauge)
}

func (ms *MemStorage) GetAllCounters() CounterMap {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return maps.Clone(ms.Counter)
}

type AgentStorage struct {
	Gauge   GaugeMap
	Counter CounterMap
	mu      *sync.RWMutex
}

func NewAgentStorage() *AgentStorage {
	return &AgentStorage{
		Gauge:   make(GaugeMap),
		Counter: make(CounterMap),
		mu:      &sync.RWMutex{},
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

func (as *AgentStorage) Snapshot() (g GaugeMap, c CounterMap) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	g = maps.Clone(as.Gauge)
	c = maps.Clone(as.Counter)
	return g, c
}
