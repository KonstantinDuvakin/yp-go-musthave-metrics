package storage

type GaugeMap map[string]float64
type CounterMap map[string]int64

type ServerStorage interface {
	SetGauge(string, float64)
	AddCounter(string, int64)
	GetGauge(string) (float64, bool)
	GetCounter(string) (int64, bool)
	GetAllGauges() GaugeMap
	GetAllCounters() CounterMap
	SaveMetricsToFile(filename string) error
	RestoreFromFile(filename string) error
}
