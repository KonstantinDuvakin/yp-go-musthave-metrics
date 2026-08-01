package getMetricJson

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage/memStorage"
)

func TestGetMetricJson_StatusCodes(t *testing.T) {
	type tc struct {
		name       string
		body       string
		wantStatus int
	}

	tests := []tc{
		{
			name:       "counter ok",
			body:       `{"id":"PollCount","type":"counter"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "gauge ok",
			body:       `{"id":"Alloc","type":"gauge"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			body:       `{"id":"Alloc","type":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "counter not found",
			body:       `{"id":"Unknown","type":"counter"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "gauge not found",
			body:       `{"id":"Unknown","type":"gauge"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unknown type",
			body:       `{"id":"Alloc","type":"histogram"}`,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := memStorage.NewMemStorage()
			ms.AddCounter("PollCount", 10)
			ms.SetGauge("Alloc", 123.45)

			h := GetMetricJson(ms)

			req := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h(rec, req)
			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status=%d, want=%d", res.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestGetMetricJson_CounterResponseBody(t *testing.T) {
	ms := memStorage.NewMemStorage()
	ms.AddCounter("PollCount", 42)

	h := GetMetricJson(ms)

	req := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(`{"id":"PollCount","type":"counter"}`))
	rec := httptest.NewRecorder()

	h(rec, req)
	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want=%d", res.StatusCode, http.StatusOK)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type=%q, want application/json", ct)
	}

	var got models.Metrics
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != "PollCount" {
		t.Errorf("id=%q, want PollCount", got.ID)
	}
	if got.MType != models.Counter {
		t.Errorf("type=%q, want counter", got.MType)
	}
	if got.Delta == nil || *got.Delta != 42 {
		t.Errorf("delta=%v, want 42", got.Delta)
	}
}

func TestGetMetricJson_GaugeResponseBody(t *testing.T) {
	ms := memStorage.NewMemStorage()
	ms.SetGauge("Alloc", 3.14)

	h := GetMetricJson(ms)

	req := httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(`{"id":"Alloc","type":"gauge"}`))
	rec := httptest.NewRecorder()

	h(rec, req)
	res := rec.Result()

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want=%d", res.StatusCode, http.StatusOK)
	}

	var got models.Metrics
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != "Alloc" {
		t.Errorf("id=%q, want Alloc", got.ID)
	}
	if got.MType != models.Gauge {
		t.Errorf("type=%q, want gauge", got.MType)
	}
	if got.Value == nil || *got.Value != 3.14 {
		t.Errorf("value=%v, want 3.14", got.Value)
	}
}
