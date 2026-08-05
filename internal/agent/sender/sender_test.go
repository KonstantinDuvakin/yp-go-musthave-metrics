package sender

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/updateHandler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/updateMetricJson"
	gzipmw "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/gzip"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage/memStorage"
	"github.com/go-chi/chi/v5"
)

const addr = "localhost:8080"

func TestSendMetrics(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "positive counter test #1",
			args: args{
				url: fmt.Sprintf("http://%s/update/counter/counter/1", addr),
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "application/json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := memStorage.NewMemStorage()

			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", updateHandler.UpdateHandler(ms))

			request := httptest.NewRequest(http.MethodPost, tt.args.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, request)

			res := w.Result()
			if got := res.StatusCode; got != tt.want.code {
				t.Errorf("SendMetrics() = %d, want %d", got, tt.want.code)
			}
			res.Body.Close()
		})
	}
}

func TestSendMetricsJson(t *testing.T) {
	delta := int64(5)

	tests := []struct {
		name string
		body models.Metrics
	}{
		{
			name: "positive counter test #1",
			body: models.Metrics{
				ID:    "PollCount",
				MType: models.Counter,
				Delta: &delta,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Content-Encoding"); got != "gzip" {
					t.Errorf("Content-Encoding = %q, want %q", got, "gzip")
				}
				if got := r.Header.Get("Content-Type"); got != "application/json" {
					t.Errorf("Content-Type = %q, want %q", got, "application/json")
				}

				zr, err := gzip.NewReader(r.Body)
				if err != nil {
					t.Fatalf("gzip.NewReader: тело не является валидным gzip: %v", err)
				}
				defer zr.Close()

				data, err := io.ReadAll(zr)
				if err != nil {
					t.Fatalf("чтение распакованного тела: %v", err)
				}

				var got models.Metrics
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatalf("распакованное тело не является валидным JSON: %v", err)
				}

				if got.ID != tt.body.ID || got.MType != tt.body.MType {
					t.Errorf("получено %+v, ожидалось %+v", got, tt.body)
				}
				if got.Delta == nil || *got.Delta != *tt.body.Delta {
					t.Errorf("Delta = %v, ожидалось %v", got.Delta, *tt.body.Delta)
				}

				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"ok"}`))
			}))
			defer srv.Close()

			s := NewSender(strings.TrimPrefix(srv.URL, "http://"))

			if err := s.SendMetricsJson(context.Background(), tt.body); err != nil {
				t.Fatalf("SendMetricsJson() вернул ошибку: %v", err)
			}
		})
	}
}

func TestSendMetricsJson_Integration(t *testing.T) {
	delta := int64(5)
	body := models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &delta}

	store := memStorage.NewMemStorage()

	r := chi.NewRouter()
	r.Use(gzipmw.Middleware)
	r.Post("/update", updateMetricJson.UpdateMetricJson(store))

	srv := httptest.NewServer(r)
	defer srv.Close()

	s := NewSender(strings.TrimPrefix(srv.URL, "http://"))

	if err := s.SendMetricsJson(context.Background(), body); err != nil {
		t.Fatalf("SendMetricsJson() вернул ошибку: %v", err)
	}

	got, ok, err := store.GetCounter(body.ID)
	if err != nil {
		t.Fatal("Ошибка получения counter")
	}
	if !ok {
		t.Fatalf("counter %q не сохранён — round-trip через gzip не сработал", body.ID)
	}
	if got != *body.Delta {
		t.Errorf("counter %q = %d, ожидалось %d", body.ID, got, *body.Delta)
	}
}

func TestUrlBuilder(t *testing.T) {
	type args struct {
		metricType string
		name       string
		value      string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty url",
			args: args{
				metricType: "",
				name:       "",
				value:      "",
			},
			want: "/update///",
		},
		{
			name: "counter url",
			args: args{
				metricType: "counter",
				name:       "counter",
				value:      "1",
			},
			want: "/update/counter/counter/1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := URLBuilder(tt.args.metricType, tt.args.name, tt.args.value); got != tt.want {
				t.Errorf("URLBuilder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendMetricsBatch(t *testing.T) {
	gv := 1.5
	var cd int64 = 10
	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &gv},
		{ID: "PollCount", MType: models.Counter, Delta: &cd},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/updates" {
			t.Errorf("path = %s, want /updates", r.URL.Path)
		}
		if got := r.Header.Get("Content-Encoding"); got != "gzip" {
			t.Errorf("Content-Encoding = %q, want gzip", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("тело не является валидным gzip: %v", err)
		}
		defer zr.Close()

		data, err := io.ReadAll(zr)
		if err != nil {
			t.Fatalf("чтение распакованного тела: %v", err)
		}

		var got []models.Metrics
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("тело не является валидным []Metrics JSON: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("получено %d метрик, ожидалось 2", len(got))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewSender(strings.TrimPrefix(srv.URL, "http://"))
	if err := s.SendMetricsBatch(context.Background(), batch); err != nil {
		t.Fatalf("SendMetricsBatch() вернул ошибку: %v", err)
	}
}

func TestSendMetricsBatch_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := NewSender(strings.TrimPrefix(srv.URL, "http://"))

	gv := 1.0
	batch := []models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &gv}}
	if err := s.SendMetricsBatch(context.Background(), batch); err == nil {
		t.Fatal("ожидалась ошибка при статусе 500, получили nil")
	}
}
