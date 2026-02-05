package handler

import (
	"net/http"
	"strconv"
	"strings"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

func UpdateHandler(storage *storage.MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 5 {
			http.Error(w, "not found resource", http.StatusNotFound)
			return
		}

		metricType := parts[2]
		metricName := parts[3]
		metricValue := parts[4]

		if metricType == "" {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		switch metricType {
		case models.Counter:
			if metricName == "" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}

			parsedValue, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}

			storage.AddCounter(metricName, parsedValue)

		case models.Gauge:
			if metricName == "" {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}

			parsedValue, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(w, "invalid path", http.StatusBadRequest)
				return
			}

			storage.SetGauge(metricName, parsedValue)
		default:
			http.Error(w, "invalid metric type", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
