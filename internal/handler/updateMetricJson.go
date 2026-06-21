package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

func UpdateMetricJson(storage *storage.MemStorage) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req models.Metrics

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&req); err != nil && errors.Is(err, io.EOF) {
			logger.Log.Error("Invalid json", zap.Error(err))
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Invalid json"))
			return
		}

		if req.ID == "" {
			logger.Log.Error("Id is omitted", zap.String("id", req.ID))
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Id is required"))
			return
		}

		switch req.MType {
		case models.Counter:
			if req.Delta == nil {
				logger.Log.Error("Delta is omitted", zap.String("type", req.MType))
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte("Delta for type \"counter\" is required"))
				return
			}
			storage.AddCounter(req.ID, *req.Delta)

		case models.Gauge:
			if req.Value == nil {
				logger.Log.Error("Value is omitted", zap.String("type", req.MType))
				rw.WriteHeader(http.StatusBadRequest)
				rw.Write([]byte("Value for type \"gauge\" is required"))
				return
			}
			storage.SetGauge(req.ID, *req.Value)

		default:
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Invalid metric type"))
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
