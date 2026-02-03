package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func GetMetricHandler(storage *storage.MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")

		if metricType == "" || metricName == "" {
			http.Error(w, "Not found metric type or metric name", http.StatusNotFound)
			return
		}

		var res string

		switch metricType {
		case models.Gauge:
			val, ok := storage.GetGauge(metricName)
			if !ok {
				http.Error(w, "Not found metric with name "+metricName, http.StatusNotFound)
				return
			}
			res = strconv.FormatFloat(val, 'g', -1, 64)
		case models.Counter:
			val, ok := storage.GetCounter(metricName)
			if !ok {
				http.Error(w, "Not found counter with name "+metricName, http.StatusNotFound)
				return
			}
			res = strconv.FormatInt(val, 10)
		default:
			http.Error(w, "Not found counter or metric with name "+metricName, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, fmt.Sprintf("%s = %s\n", metricName, res))
	}
}
