package repository

type MemStorage struct {
	Gauge   map[string]float64
	Counter map[string]int64
}

func (ms *MemStorage) SetGauge(field string, value float64) {
	ms.Gauge[field] = value
}

func (ms *MemStorage) AddCounter(field string, value int64) {
	ms.Counter[field] += value
}
