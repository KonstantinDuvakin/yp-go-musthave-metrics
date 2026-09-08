package memstorage

import (
	"context"
	"fmt"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
)

// Базовые операции с in-memory хранилищем: gauge перезаписывается, counter
// суммируется.
func ExampleMemStorage() {
	s := NewMemStorage()

	_ = s.SetGauge("Alloc", 42.5)
	_ = s.AddCounter("PollCount", 1)
	_ = s.AddCounter("PollCount", 4)

	gauge, _, _ := s.GetGauge("Alloc")
	counter, _, _ := s.GetCounter("PollCount")

	fmt.Println("Alloc:", gauge)
	fmt.Println("PollCount:", counter)

	// Output:
	// Alloc: 42.5
	// PollCount: 5
}

// Пакетное сохранение метрик одним вызовом.
func ExampleMemStorage_SaveMetricsBatch() {
	s := NewMemStorage()

	value := 42.5
	delta := int64(10)
	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &value},
		{ID: "PollCount", MType: models.Counter, Delta: &delta},
	}

	if err := s.SaveMetricsBatch(context.Background(), batch); err != nil {
		fmt.Println("ошибка:", err)
		return
	}

	gauge, _, _ := s.GetGauge("Alloc")
	counter, _, _ := s.GetCounter("PollCount")
	fmt.Println(gauge, counter)

	// Output:
	// 42.5 10
}
