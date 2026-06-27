package handler

import (
	"encoding/json"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

func GetMetricJson(storage *storage.MemStorage) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req models.Metrics

		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			logger.Log.Error("error decoding json: %v", zap.Error(err))
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Invalid json"))
			return
		}

		switch req.MType {
		case models.Counter:
			val, ok := storage.GetCounter(req.ID)
			if !ok {
				logger.Log.Error("error getting metric value", zap.String("id", req.ID))
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte("No such counter metric"))
				return
			}

			req.Delta = &val
		case models.Gauge:
			val, ok := storage.GetGauge(req.ID)
			if !ok {
				logger.Log.Error("error getting metric value", zap.String("id", req.ID))
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte("No such gauge metric"))
				return
			}

			req.Value = &val
		default:
			logger.Log.Error("unknown metric type", zap.String("type", req.MType))
			rw.WriteHeader(http.StatusNotFound)
			rw.Write([]byte("Unknown metric type"))
			return
		}

		enc := json.NewEncoder(rw)

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)

		if err := enc.Encode(req); err != nil {
			logger.Log.Error("error encoding json: ", zap.Error(err))
			rw.Write([]byte("Error encoding json"))
			return
		}
	}
}
