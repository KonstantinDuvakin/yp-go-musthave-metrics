package collector

import "testing"

// BenchmarkCollector измеряет стоимость сбора метрик рантайма Go.
func BenchmarkCollector(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Collector()
	}
}
