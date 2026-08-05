package updateMetricJson

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage/memStorage"
)

func TestUpdateMetricJson_StatusCodes(t *testing.T) {
	type tc struct {
		name       string
		body       string
		wantStatus int
	}

	tests := []tc{
		{
			name:       "counter ok",
			body:       `{"id":"PollCount","type":"counter","delta":10}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "gauge ok",
			body:       `{"id":"Alloc","type":"gauge","value":123.45}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid json",
			body:       `{"id":"Alloc","type": "counter","delta":10`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing id",
			body:       `{"type":"counter","delta":10}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing type",
			body:       `{"id":"Alloc","delta":10}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown type",
			body:       `{"id":"Alloc","type":"histogram","value":1}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "counter without delta",
			body:       `{"id":"PollCount","type":"counter"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "gauge without value",
			body:       `{"id":"Alloc","type":"gauge"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := memStorage.NewMemStorage()
			h := UpdateMetricJson(ms)

			req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(tt.body))
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

func TestUpdateMetricJson_StoresGauge(t *testing.T) {
	ms := memStorage.NewMemStorage()
	h := UpdateMetricJson(ms)

	body := `{"id":"Alloc","type":"gauge","value":42.5}`
	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want=%d", rec.Code, http.StatusOK)
	}

	got, ok, _ := ms.GetGauge("Alloc")
	if !ok {
		t.Fatal("gauge Alloc not found in storage")
	}
	if got != 42.5 {
		t.Fatalf("gauge value=%v, want=42.5", got)
	}
}

func TestUpdateMetricJson_CounterAccumulates(t *testing.T) {
	ms := memStorage.NewMemStorage()
	h := UpdateMetricJson(ms)

	send := func(delta int64) {
		body := `{"id":"PollCount","type":"counter","delta":` + strconv.FormatInt(delta, 10) + `}`
		req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d, want=%d", rec.Code, http.StatusOK)
		}
	}

	send(10)
	send(5)

	got, ok, _ := ms.GetCounter("PollCount")
	if !ok {
		t.Fatal("counter PollCount not found in storage")
	}
	if got != 15 {
		t.Fatalf("counter value=%d, want=15", got)
	}
}
