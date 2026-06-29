package memStorage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndRestore_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")

	src := NewMemStorage()
	src.AddCounter("PollCount", 5)
	src.SetGauge("Alloc", 3.5)

	if err := src.SaveMetricsToFile(path); err != nil {
		t.Fatalf("SaveMetricsToFile: %v", err)
	}

	dst := NewMemStorage()
	if err := dst.RestoreFromFile(path); err != nil {
		t.Fatalf("RestoreFromFile: %v", err)
	}

	if got, ok := dst.GetCounter("PollCount"); !ok || got != 5 {
		t.Errorf("counter PollCount = %d, ok=%v; want 5, true", got, ok)
	}
	if got, ok := dst.GetGauge("Alloc"); !ok || got != 3.5 {
		t.Errorf("gauge Alloc = %v, ok=%v; want 3.5, true", got, ok)
	}
}

func TestSaveMetricsToFile_WritesToGivenPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")

	ms := NewMemStorage()
	ms.AddCounter("c", 1)

	if err := ms.SaveMetricsToFile(path); err != nil {
		t.Fatalf("SaveMetricsToFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("файл не создан по пути %q: %v", path, err)
	}
	if info.Size() == 0 {
		t.Error("файл пустой, ожидались данные метрик")
	}
}

func TestRestoreFromFile_MissingFile(t *testing.T) {
	ms := NewMemStorage()

	err := ms.RestoreFromFile(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err == nil {
		t.Fatal("ожидалась ошибка при restore из несуществующего файла, получили nil")
	}
}

func TestSyncMemStorage_PersistsAfterClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.json")

	store := NewSyncMemStorage(NewMemStorage(), path)
	store.AddCounter("PollCount", 7)
	store.SetGauge("Alloc", 2.5)

	store.Close()

	restored := NewMemStorage()
	if err := restored.RestoreFromFile(path); err != nil {
		t.Fatalf("RestoreFromFile после Close: %v", err)
	}

	if got, ok := restored.GetCounter("PollCount"); !ok || got != 7 {
		t.Errorf("counter PollCount = %d, ok=%v; want 7, true", got, ok)
	}
	if got, ok := restored.GetGauge("Alloc"); !ok || got != 2.5 {
		t.Errorf("gauge Alloc = %v, ok=%v; want 2.5, true", got, ok)
	}
}
