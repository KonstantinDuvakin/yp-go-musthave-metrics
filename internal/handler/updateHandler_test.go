package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/memStorage"
	"github.com/go-chi/chi/v5"
)

func newTestRouter(ms *memStorage.MemStorage) http.Handler {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", UpdateHandler(ms))
	return r
}

func TestUpdateHandler_StatusCodes(t *testing.T) {
	type tc struct {
		name       string
		method     string
		path       string
		wantStatus int
	}

	tests := []tc{
		{
			name:       "counter ok",
			method:     http.MethodPost,
			path:       "/update/" + models.Counter + "/PollCount/10",
			wantStatus: http.StatusOK,
		},
		{
			name:       "gauge ok",
			method:     http.MethodPost,
			path:       "/update/" + models.Gauge + "/Alloc/123.45",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid metric type",
			method:     http.MethodPost,
			path:       "/update/unknown/Alloc/1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "counter invalid value",
			method:     http.MethodPost,
			path:       "/update/" + models.Counter + "/PollCount/notint",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "gauge invalid value",
			method:     http.MethodPost,
			path:       "/update/" + models.Gauge + "/Alloc/notfloat",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong method",
			method:     http.MethodGet,
			path:       "/update/" + models.Gauge + "/Alloc/1",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "not found route (missing value segment)",
			method:     http.MethodPost,
			path:       "/update/" + models.Gauge + "/Alloc",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := memStorage.NewMemStorage()
			router := newTestRouter(ms)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)
			res := rec.Result()

			if res.StatusCode != tt.wantStatus {
				t.Fatalf("status=%d, want=%d", res.StatusCode, tt.wantStatus)
			}

			res.Body.Close()
		})
	}
}
