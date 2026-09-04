package root_handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

func TestRootHandler_OK_EmptyStorage(t *testing.T) {
	ms := mem_storage.NewMemStorage()

	h := RootHandler(ms)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want=%d", res.StatusCode, http.StatusOK)
	}

	ct := res.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type=%q, want prefix %q", ct, "text/html")
	}

	bodyBytes, _ := io.ReadAll(res.Body)
	body := string(bodyBytes)

	if !strings.Contains(body, "<html>") || !strings.Contains(body, "</html>") {
		t.Fatalf("body does not look like html: %q", body)
	}
	if !strings.Contains(body, "<h1>Metrics</h1>") {
		t.Fatalf("missing header in body: %q", body)
	}
}

func TestRootHandler_OK_WithMetrics(t *testing.T) {
	ms := mem_storage.NewMemStorage()

	ms.SetGauge("Alloc", 123.5)
	ms.SetGauge("RandomValue", 0.25)
	ms.AddCounter("PollCount", 10)
	ms.AddCounter("PollCount", 2)

	h := RootHandler(ms)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want=%d", res.StatusCode, http.StatusOK)
	}

	bodyBytes, _ := io.ReadAll(res.Body)
	body := string(bodyBytes)

	if !strings.Contains(body, "<li>Alloc: 123.500000</li>") {
		t.Fatalf("missing Alloc metric in body: %q", body)
	}
	if !strings.Contains(body, "<li>RandomValue: 0.250000</li>") {
		t.Fatalf("missing RandomValue metric in body: %q", body)
	}

	// Counter печатается как %d
	if !strings.Contains(body, "<li>PollCount: 12</li>") {
		t.Fatalf("missing PollCount metric in body: %q", body)
	}
}
