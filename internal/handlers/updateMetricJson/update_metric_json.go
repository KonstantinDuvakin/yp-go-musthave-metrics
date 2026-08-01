package updateMetricJson

import (
	"encoding/json"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

func UpdateMetricJson(storage storage.MetricsStorage) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req models.Metrics

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&req); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Invalid json"))
			return
		}

		if req.ID == "" {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Id is required"))
			return
		}

		switch req.MType {
		case models.Counter:
			if req.Delta == nil {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte("Delta for type \"counter\" is required"))
				return
			}
			err := storage.AddCounter(req.ID, *req.Delta)
			if err != nil {
				logger.Log.Error("Error adding counter", zap.Error(err))
				http.Error(rw, "internal error", http.StatusInternalServerError)
				return
			}

		case models.Gauge:
			if req.Value == nil {
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte("Value for type \"gauge\" is required"))
				return
			}
			err := storage.SetGauge(req.ID, *req.Value)
			if err != nil {
				logger.Log.Error("Error setting gauge", zap.Error(err))
				http.Error(rw, "internal error", http.StatusInternalServerError)
				return
			}

		default:
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Invalid metric type"))
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
