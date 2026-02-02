package collector

import (
	"math"
	"testing"
)

func TestCollector(t *testing.T) {
	got := Collector()

	requiredKeys := []string{
		"Alloc",
		"BuckHashSys",
		"Frees",
		"GCCPUFraction",
		"GCSys",
		"HeapAlloc",
		"HeapIdle",
		"HeapInuse",
		"HeapObjects",
		"HeapReleased",
		"HeapSys",
		"LastGC",
		"Lookups",
		"MCacheInuse",
		"MCacheSys",
		"MSpanInuse",
		"MSpanSys",
		"Mallocs",
		"NextGC",
		"NumForcedGC",
		"NumGC",
		"OtherSys",
		"PauseTotalNs",
		"StackInuse",
		"StackSys",
		"Sys",
		"TotalAlloc",
		"RandomValue",
	}

	for _, key := range requiredKeys {
		if _, ok := got[key]; !ok {
			t.Errorf("Collector(): missing key %q", key)
		}
	}
}

func TestCollector_RandomValue(t *testing.T) {
	got := Collector()

	v, ok := got["RandomValue"]
	if !ok {
		t.Fatal("RandomValue not found")
	}

	if v < 0 || v >= 1 {
		t.Errorf("RandomValue = %v, want value in [0,1)", v)
	}
}

func TestCollector_NoNaN(t *testing.T) {
	got := Collector()

	for k, v := range got {
		if math.IsNaN(v) {
			t.Errorf("Collector(): value for %s is NaN", k)
		}
	}
}
