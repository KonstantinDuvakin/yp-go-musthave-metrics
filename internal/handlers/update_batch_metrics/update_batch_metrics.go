// Package updatebatchmetrics содержит HTTP-обработчик пакетного обновления
// метрик и записи события аудита о принятом пакете.
package updatebatchmetrics

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

// UpdateBatchMetrics возвращает обработчик POST /updates, сохраняющий пакет
// метрик из JSON-тела запроса (срез [models.Metrics]) одним вызовом.
//
// После успешного сохранения вызывает auditFunc с событием [models.Audit],
// содержащим время, имена метрик и адрес отправителя. Отвечает 200 при
// успехе, 400 при некорректном JSON, 500 при ошибке хранилища.
func UpdateBatchMetrics(store storage.MetricsStorage, auditFunc func(event models.Audit)) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req []models.Metrics

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&req); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Invalid json"))
			return
		}

		if err := store.SaveMetricsBatch(r.Context(), req); err != nil {
			logger.Log.Error("Failed to save metrics batch: ", zap.Error(err))
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte("Couldn't save metrics batch"))
			return
		}

		event := models.Audit{
			TS:        time.Now().UnixMilli(),
			Metrics:   make([]string, 0, len(req)),
			IPAddress: r.RemoteAddr,
		}

		for _, el := range req {
			event.Metrics = append(event.Metrics, el.ID)
		}

		auditFunc(event)

		rw.WriteHeader(http.StatusOK)
	}
}
