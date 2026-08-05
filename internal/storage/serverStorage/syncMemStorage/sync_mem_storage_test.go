package syncMemStorage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage/memStorage"
)

func TestSaveAndRestore_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")

	src := memStorage.NewMemStorage()
	src.AddCounter("PollCount", 5)
	src.SetGauge("Alloc", 3.5)

	if err := src.SaveMetricsToFile(path); err != nil {
		t.Fatalf("SaveMetricsToFile: %v", err)
	}

	dst := memStorage.NewMemStorage()
	if err := dst.RestoreFromFile(path); err != nil {
		t.Fatalf("RestoreFromFile: %v", err)
	}

	if got, ok, _ := dst.GetCounter("PollCount"); !ok || got != 5 {
		t.Errorf("counter PollCount = %d, ok=%v; want 5, true", got, ok)
	}
	if got, ok, _ := dst.GetGauge("Alloc"); !ok || got != 3.5 {
		t.Errorf("gauge Alloc = %v, ok=%v; want 3.5, true", got, ok)
	}
}

func TestSaveMetricsToFile_WritesToGivenPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")

	ms := memStorage.NewMemStorage()
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
	ms := memStorage.NewMemStorage()

	err := ms.RestoreFromFile(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err == nil {
		t.Fatal("ожидалась ошибка при restore из несуществующего файла, получили nil")
	}
}

func TestSyncMemStorage_PersistsAfterClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.json")

	store := NewSyncMemStorage(memStorage.NewMemStorage(), path)
	store.AddCounter("PollCount", 7)
	store.SetGauge("Alloc", 2.5)

	store.Close()

	restored := memStorage.NewMemStorage()
	if err := restored.RestoreFromFile(path); err != nil {
		t.Fatalf("RestoreFromFile после Close: %v", err)
	}

	if got, ok, _ := restored.GetCounter("PollCount"); !ok || got != 7 {
		t.Errorf("counter PollCount = %d, ok=%v; want 7, true", got, ok)
	}
	if got, ok, _ := restored.GetGauge("Alloc"); !ok || got != 2.5 {
		t.Errorf("gauge Alloc = %v, ok=%v; want 2.5, true", got, ok)
	}
}
