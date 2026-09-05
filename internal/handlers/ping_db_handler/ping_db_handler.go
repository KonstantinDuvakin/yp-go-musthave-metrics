// Package pingdbhandler содержит HTTP-обработчик проверки доступности БД.
package pingdbhandler

import (
	"context"
	"net/http"
)

// Pinger — источник данных, доступность которого можно проверить. Ему
// удовлетворяет, в частности, пул соединений pgx.
type Pinger interface {
	Ping(ctx context.Context) error
}

// PingDBHandler возвращает обработчик GET /ping, проверяющий соединение с БД.
//
// Отвечает 200 и телом "OK", если pinger успешно отвечает на Ping; 500 —
// если pinger равен nil или проверка завершилась ошибкой.
func PingDBHandler(pinger Pinger) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if pinger == nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := pinger.Ping(r.Context()); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("OK"))
	}
}
