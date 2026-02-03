package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func newValueRouter(ms *storage.MemStorage) http.Handler {
	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", GetMetricHandler(ms))
	return r
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestGetMetricHandler_GaugeOK(t *testing.T) {
	ms := storage.NewMemStorage()
	ms.SetGauge("Alloc", 123.5)

	router := newValueRouter(ms)

	req := httptest.NewRequest(http.MethodGet, "/value/"+models.Gauge+"/Alloc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want=%d", res.StatusCode, http.StatusOK)
	}

	ct := res.Header.Get("Content-Type")
	// у тебя ставится "text/plain" (без charset)
	if !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("Content-Type=%q, want prefix %q", ct, "text/plain")
	}

	body := readBody(t, res)
	// Ты отдаёшь: "%s = %s\n"
	if body != "Alloc = 123.5\n" && body != "Alloc = 123.500000\n" {
		// Формат 'g' даст 123.5, но если поменяешь формат — тест покажет.
		t.Fatalf("body=%q, want %q", body, "Alloc = 123.5\\n")
	}
}

func TestGetMetricHandler_CounterOK(t *testing.T) {
	ms := storage.NewMemStorage()
	ms.AddCounter("PollCount", 10)
	ms.AddCounter("PollCount", 2)

	router := newValueRouter(ms)

	req := httptest.NewRequest(http.MethodGet, "/value/"+models.Counter+"/PollCount", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want=%d", res.StatusCode, http.StatusOK)
	}

	body := readBody(t, res)
	if body != "PollCount = 12\n" {
		t.Fatalf("body=%q, want %q", body, "PollCount = 12\\n")
	}
}

func TestGetMetricHandler_NotFoundMetric(t *testing.T) {
	ms := storage.NewMemStorage()
	router := newValueRouter(ms)

	req := httptest.NewRequest(http.MethodGet, "/value/"+models.Gauge+"/Unknown", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d, want=%d", res.StatusCode, http.StatusNotFound)
	}
}

func TestGetMetricHandler_RouteNotMatched(t *testing.T) {
	ms := storage.NewMemStorage()
	router := newValueRouter(ms)

	// не хватает сегмента {name} -> chi вернёт 404 до handler
	req := httptest.NewRequest(http.MethodGet, "/value/"+models.Gauge, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d, want=%d", res.StatusCode, http.StatusNotFound)
	}
}

func TestGetMetricHandler_InvalidType_ShouldBeNotFoundOrBadRequest(t *testing.T) {
	ms := storage.NewMemStorage()
	ms.SetGauge("Alloc", 1)

	router := newValueRouter(ms)

	req := httptest.NewRequest(http.MethodGet, "/value/unknown/Alloc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	// В ИДЕАЛЕ: 400 Bad Request или 404 Not Found, но главное — НЕ 200 OK.
	// Сейчас твой handler делает http.Error(...) но НЕ return, а потом пишет 200 OK.
	// Поэтому этот тест, скорее всего, у тебя сейчас УПАДЁТ (и это сигнал исправить default).
	if res.StatusCode == http.StatusOK {
		body := readBody(t, res)
		t.Fatalf("got status 200 with body=%q, want non-200 for invalid type", body)
	}
}
