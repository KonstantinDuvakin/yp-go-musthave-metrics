package memstorage

import (
	"context"
	"fmt"
	"testing"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
)

// BenchmarkMemStorage_SetGauge измеряет запись gauge-метрики.
func BenchmarkMemStorage_SetGauge(b *testing.B) {
	s := NewMemStorage()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = s.SetGauge("Alloc", float64(i))
	}
}

// BenchmarkMemStorage_AddCounter измеряет накопление counter-метрики.
func BenchmarkMemStorage_AddCounter(b *testing.B) {
	s := NewMemStorage()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = s.AddCounter("PollCount", 1)
	}
}

// BenchmarkMemStorage_SaveMetricsBatch измеряет пакетное сохранение метрик.
func BenchmarkMemStorage_SaveMetricsBatch(b *testing.B) {
	s := NewMemStorage()

	value := 42.5
	delta := int64(1)
	batch := make([]models.Metrics, 0, 40)
	for i := 0; i < 20; i++ {
		batch = append(batch,
			models.Metrics{ID: fmt.Sprintf("g%d", i), MType: models.Gauge, Value: &value},
			models.Metrics{ID: fmt.Sprintf("c%d", i), MType: models.Counter, Delta: &delta},
		)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = s.SaveMetricsBatch(context.Background(), batch)
	}
}

// BenchmarkMemStorage_GetAllGauges измеряет получение копии всех gauge-метрик.
func BenchmarkMemStorage_GetAllGauges(b *testing.B) {
	s := NewMemStorage()
	for i := 0; i < 100; i++ {
		_ = s.SetGauge(fmt.Sprintf("g%d", i), float64(i))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = s.GetAllGauges()
	}
}
