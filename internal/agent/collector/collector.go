// Package collector собирает метрики среды выполнения: показатели рантайма
// Go и данные о памяти и загрузке CPU системы.
package collector

import (
	"fmt"
	"math/rand"
	"runtime"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// Collector собирает метрики рантайма Go через runtime.ReadMemStats и
// возвращает их как мапу имя -> значение. Дополнительно добавляет
// случайную метрику RandomValue.
func Collector() map[string]float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]float64{
		"Alloc":         float64(m.Alloc),
		"BuckHashSys":   float64(m.BuckHashSys),
		"Frees":         float64(m.Frees),
		"GCCPUFraction": m.GCCPUFraction,
		"GCSys":         float64(m.GCSys),
		"HeapAlloc":     float64(m.HeapAlloc),
		"HeapIdle":      float64(m.HeapIdle),
		"HeapInuse":     float64(m.HeapInuse),
		"HeapObjects":   float64(m.HeapObjects),
		"HeapReleased":  float64(m.HeapReleased),
		"HeapSys":       float64(m.HeapSys),
		"LastGC":        float64(m.LastGC),
		"Lookups":       float64(m.Lookups),
		"MCacheInuse":   float64(m.MCacheInuse),
		"MCacheSys":     float64(m.MCacheSys),
		"MSpanInuse":    float64(m.MSpanInuse),
		"MSpanSys":      float64(m.MSpanSys),
		"Mallocs":       float64(m.Mallocs),
		"NextGC":        float64(m.NextGC),
		"NumForcedGC":   float64(m.NumForcedGC),
		"NumGC":         float64(m.NumGC),
		"OtherSys":      float64(m.OtherSys),
		"PauseTotalNs":  float64(m.PauseTotalNs),
		"StackInuse":    float64(m.StackInuse),
		"StackSys":      float64(m.StackSys),
		"Sys":           float64(m.Sys),
		"TotalAlloc":    float64(m.TotalAlloc),
		"RandomValue":   rand.Float64(),
	}
}

// GopsCollector собирает метрики системы через gopsutil: общий и свободный
// объём памяти и загрузку каждого логического CPU (CPUutilizationN).
// Возвращает их как мапу имя -> значение.
func GopsCollector() map[string]float64 {
	v, _ := mem.VirtualMemory()
	cpus, _ := cpu.Percent(0, true)

	res := map[string]float64{
		"TotalMemory": float64(v.Total),
		"FreeMemory":  float64(v.Free),
	}

	for i, c := range cpus {
		name := fmt.Sprintf("CPUutilization%d", i+1)
		res[name] = c
	}

	return res
}
