package handler

import (
	"net/http"
	"strconv"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func UpdateHandler(storage storage.ServerStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		metricValue := chi.URLParam(r, "value")
		if metricType == "" {
			http.Error(w, "Invalid path. Absent metric type", http.StatusBadRequest)
			return
		}

		switch metricType {
		case models.Counter:
			if metricName == "" {
				http.Error(w, "Not found counter with name "+metricName, http.StatusNotFound)
				return
			}

			parsedValue, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(w, "Invalid value for counter "+metricName, http.StatusBadRequest)
				return
			}

			storage.AddCounter(metricName, parsedValue)

		case models.Gauge:
			if metricName == "" {
				http.Error(w, "Not found metric with name "+metricName, http.StatusNotFound)
				return
			}

			parsedValue, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(w, "Invalid value for metric "+metricName, http.StatusBadRequest)
				return
			}

			storage.SetGauge(metricName, parsedValue)
		default:
			http.Error(w, "Invalid metric type, name or value", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
