// Package getmetricjson содержит HTTP-обработчик получения значения метрики
// по JSON-запросу.
package getmetricjson

import (
	"encoding/json"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

// GetMetricJSON возвращает обработчик POST /value, принимающий [models.Metrics]
// с полями ID и MType и отдающий эту же метрику с заполненным значением
// (Delta или Value) в формате JSON.
//
// Отвечает 200 и метрикой при успехе, 400 при некорректном JSON, 404 при
// отсутствии метрики или неизвестном типе, 500 при ошибке хранилища.
func GetMetricJSON(storage storage.MetricsStorage) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req models.Metrics

		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Invalid json"))
			return
		}

		switch req.MType {
		case models.Counter:
			val, ok, err := storage.GetCounter(req.ID)
			if err != nil {
				logger.Log.Error("error getting metric value", zap.String("id", req.ID))
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte("Error getting metric value"))
				return
			}

			if !ok {
				logger.Log.Error("error getting metric value", zap.String("id", req.ID))
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte("No such counter metric"))
				return
			}

			req.Delta = &val
		case models.Gauge:
			val, ok, err := storage.GetGauge(req.ID)
			if err != nil {
				logger.Log.Error("error getting metric value", zap.String("id", req.ID))
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte("Error getting metric value"))
				return
			}

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
