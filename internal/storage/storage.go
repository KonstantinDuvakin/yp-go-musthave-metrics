package storage

type GaugeMap map[string]float64
type CounterMap map[string]int64

type MetricsStorage interface {
	SetGauge(string, float64) error
	AddCounter(string, int64) error
	GetGauge(string) (float64, bool, error)
	GetCounter(string) (int64, bool, error)
	GetAllGauges() (GaugeMap, error)
	GetAllCounters() (CounterMap, error)
}

type FilePersistentStorage interface {
	SaveMetricsToFile(filename string) error
	RestoreFromFile(filename string) error
}

type ServerStorage interface {
	MetricsStorage
	FilePersistentStorage
}
