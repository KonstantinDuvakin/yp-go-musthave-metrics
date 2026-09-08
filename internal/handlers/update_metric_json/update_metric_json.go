// Package updatemetricjson содержит HTTP-обработчик обновления одной метрики,
// переданной в теле запроса в формате JSON.
package updatemetricjson

import (
	"encoding/json"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

// UpdateMetricJSON возвращает обработчик POST /update, сохраняющий одну
// метрику из JSON-тела запроса ([models.Metrics]).
//
// Для counter обязательно поле Delta, для gauge — Value. Отвечает 200 при
// успехе, 400 при некорректном JSON, пустом ID, отсутствующем значении или
// неизвестном типе, 500 при ошибке хранилища.
func UpdateMetricJSON(storage storage.MetricsStorage) http.HandlerFunc {
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
