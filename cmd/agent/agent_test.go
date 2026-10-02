package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/agent_storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// batchRecorder — тестовый сервер, запоминающий пакеты, пришедшие на /updates.
type batchRecorder struct {
	mu      sync.Mutex
	batches [][]models.Metrics
}

func (br *batchRecorder) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	zr, err := gzip.NewReader(r.Body)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	var batch []models.Metrics
	if err = json.NewDecoder(zr).Decode(&batch); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	br.mu.Lock()
	br.batches = append(br.batches, batch)
	br.mu.Unlock()
}

func (br *batchRecorder) all() [][]models.Metrics {
	br.mu.Lock()
	defer br.mu.Unlock()
	return append([][]models.Metrics(nil), br.batches...)
}

func newTestAgent(t *testing.T, h http.Handler) (*Agent, *agentstorage.AgentStorage) {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	store := agentstorage.NewAgentStorage()
	return New(store, sender.NewSender(strings.TrimPrefix(ts.URL, "http://"))), store
}

// runAgent запускает Run в горутине и возвращает канал с его результатом.
func runAgent(a *Agent, ctx, sendCtx context.Context, poll, report time.Duration) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Run(ctx, sendCtx, poll, report, "", 2, nil)
	}()
	return errCh
}

func hasMetric(batch []models.Metrics, id string) bool {
	for _, m := range batch {
		if m.ID == id {
			return true
		}
	}
	return false
}

func TestRun_SendsFinalSnapshotOnStop(t *testing.T) {
	rec := &batchRecorder{}
	agent, store := newTestAgent(t, rec)

	ctx, cancel := context.WithCancel(context.Background())
	// report больше длительности теста: по тикеру ничего не отправится,
	// единственный пакет — финальный снимок при остановке.
	errCh := runAgent(agent, ctx, context.Background(), 10*time.Millisecond, time.Hour)

	require.Eventually(t, func() bool {
		_, counters := store.Snapshot()
		return counters["PollCount"] > 0
	}, 2*time.Second, 10*time.Millisecond, "агент должен успеть опросить метрики")

	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Run не завершился после отмены ctx")
	}

	batches := rec.all()
	require.Len(t, batches, 1, "при остановке должен уйти ровно один финальный пакет")
	assert.True(t, hasMetric(batches[0], "PollCount"), "финальный пакет должен содержать PollCount")
}

func TestRun_StopsWithoutSendingWhenStorageEmpty(t *testing.T) {
	rec := &batchRecorder{}
	agent, _ := newTestAgent(t, rec)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // остановка до первого опроса

	select {
	case err := <-runAgent(agent, ctx, context.Background(), time.Hour, time.Hour):
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Run с пустым хранилищем должен завершаться сразу")
	}

	assert.Empty(t, rec.all(), "пустое хранилище не должно отправляться")
}

func TestRun_SendCtxAbortsPendingRequests(t *testing.T) {
	release := make(chan struct{})
	requested := make(chan struct{}, 1)

	// Сервер «висит», пока запрос не отменит клиент или не закончится тест.
	hang := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		select {
		case requested <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	agent, _ := newTestAgent(t, hang)
	t.Cleanup(func() { close(release) }) // выполняется до ts.Close

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sendCtx, cancelSend := context.WithCancel(context.Background())
	defer cancelSend()

	errCh := runAgent(agent, ctx, sendCtx, 10*time.Millisecond, 20*time.Millisecond)

	select {
	case <-requested:
	case <-time.After(2 * time.Second):
		t.Fatal("агент не начал отправку")
	}

	cancel()

	select {
	case <-errCh:
		t.Fatal("Run не должен завершаться, пока sendCtx активен и запросы не досланы")
	case <-time.After(200 * time.Millisecond):
	}

	cancelSend()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Run не завершился после отмены sendCtx")
	}
}
