// Package updatebatchmetrics содержит HTTP-обработчик пакетного обновления
// метрик. Сохранение и аудит выполняет сервисный слой (updatemetrics),
// обработчик только разбирает JSON и передаёт пакет ему.
package updatebatchmetrics

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

// batchUpdater — то, что нужно обработчику от сервисного слоя: принять
// пакет метрик от отправителя ip. Реализуется updatemetrics.Service.
type batchUpdater interface {
	UpdateBatch(ctx context.Context, metrics []models.Metrics, ip string) error
}

// UpdateBatchMetrics возвращает обработчик POST /updates. Он разбирает
// JSON-тело запроса (срез [models.Metrics]) и передаёт пакет в
// batchService.UpdateBatch вместе с IP отправителя (см. clientIP);
// сохранение и запись аудита выполняет сервис.
//
// Отвечает 200 при успехе, 400 при некорректном JSON (сервис при этом не
// вызывается), 500 при ошибке сервиса.
func UpdateBatchMetrics(batchService batchUpdater) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req []models.Metrics

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&req); err != nil {
			http.Error(rw, "Invalid json", http.StatusBadRequest)
			return
		}

		ip := clientIP(r)

		err := batchService.UpdateBatch(r.Context(), req, ip)
		if err != nil {
			logger.Log.Error("Не удалось сохранить метрики", zap.Error(err))
			http.Error(rw, "Couldn't save metrics batch", http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

// clientIP возвращает IP отправителя для аудита: значение заголовка
// X-Real-IP, который выставляет агент, в нормализованной записи
// (net.ParseIP), а если заголовка нет или он некорректен — адрес соединения
// (r.RemoteAddr) без порта. Адрес без порта возвращается как есть.
//
// То же правило, что у gRPC-версии в пакете metrics. Функция не возвращает
// ошибку и ничего не логирует: отсутствие или некорректность заголовка —
// штатный случай.
func clientIP(r *http.Request) string {
	ipRaw := r.Header.Get("X-Real-IP")

	if ipRaw != "" {
		if ip := net.ParseIP(ipRaw); ip != nil {
			return ip.String()
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
