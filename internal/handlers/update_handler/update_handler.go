// Package updatehandler содержит HTTP-обработчик обновления одной метрики,
// переданной в параметрах URL.
package updatehandler

import (
	"net/http"
	"strconv"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// UpdateHandler возвращает обработчик POST /update/{type}/{name}/{value},
// сохраняющий одну метрику в storage.
//
// Тип, имя и значение берутся из параметров пути. Отвечает 200 при успехе,
// 400 при неверном типе или значении, 404 при отсутствии имени и 500 при
// ошибке хранилища.
func UpdateHandler(storage storage.MetricsStorage) http.HandlerFunc {
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

			err = storage.AddCounter(metricName, parsedValue)
			if err != nil {
				logger.Log.Error("error adding counter", zap.Error(err))
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
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

			err = storage.SetGauge(metricName, parsedValue)
			if err != nil {
				logger.Log.Error("error setting gauge", zap.Error(err))
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		default:
			http.Error(w, "Invalid metric type, name or value", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
