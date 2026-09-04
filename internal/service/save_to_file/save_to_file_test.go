package savetofile

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

func TestSaveToFile_ZeroInterval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	c := &config.ServerConfig{StoreInterval: 0, FileStoragePath: path}

	store := memstorage.NewMemStorage()
	store.AddCounter("c", 1)

	done := make(chan struct{})
	SaveToFile(context.Background(), c, store, done)

	select {
	case <-done:
	default:
		t.Fatal("done не закрыт после возврата SaveToFile при interval==0")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("при interval==0 SaveToFile не должна писать файл (err=%v)", err)
	}
}

func TestSaveToFile_GracefulSaveOnCancel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	c := &config.ServerConfig{StoreInterval: 3600, FileStoragePath: path}

	store := memstorage.NewMemStorage()
	store.AddCounter("PollCount", 42)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go SaveToFile(ctx, c, store, done)

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SaveToFile не завершилась после отмены контекста")
	}

	check := memstorage.NewMemStorage()
	if err := check.RestoreFromFile(path); err != nil {
		t.Fatalf("RestoreFromFile: %v", err)
	}
	if got, ok, _ := check.GetCounter("PollCount"); !ok || got != 42 {
		t.Errorf("после отмены: counter = %d, ok=%v; want 42, true", got, ok)
	}
}

func TestSaveToFile_TickerWritesPeriodically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	c := &config.ServerConfig{StoreInterval: 1, FileStoragePath: path}

	store := memstorage.NewMemStorage()
	store.AddCounter("PollCount", 99)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go SaveToFile(ctx, c, store, done)

	// Ждём, пока тикер (1с) создаст файл, поллим до 3с, чтобы не зависеть от точного тайминга.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			cancel()
			<-done
			return // файл появился — тикер сработал
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("тикер не записал файл за отведённое время")
}
