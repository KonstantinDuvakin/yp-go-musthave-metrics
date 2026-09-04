package get_metric_handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
	"github.com/go-chi/chi/v5"
)

func newValueRouter(ms *mem_storage.MemStorage) http.Handler {
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
	ms := mem_storage.NewMemStorage()
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
	if !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("Content-Type=%q, want prefix %q", ct, "text/plain")
	}

	body := readBody(t, res)
	if body != "123.5" && body != "123.5\n" {
		t.Fatalf("body=%q, want %q", body, "Alloc = 123.5\n")
	}
}

func TestGetMetricHandler_CounterOK(t *testing.T) {
	ms := mem_storage.NewMemStorage()
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
	if body != "12" {
		t.Fatalf("body=%q, want %q", body, "12")
	}
}

func TestGetMetricHandler_NotFoundMetric(t *testing.T) {
	ms := mem_storage.NewMemStorage()
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
	ms := mem_storage.NewMemStorage()
	router := newValueRouter(ms)

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
	ms := mem_storage.NewMemStorage()
	ms.SetGauge("Alloc", 1)

	router := newValueRouter(ms)

	req := httptest.NewRequest(http.MethodGet, "/value/unknown/Alloc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		body := readBody(t, res)
		t.Fatalf("got status 200 with body=%q, want non-200 for invalid type", body)
	}
}
