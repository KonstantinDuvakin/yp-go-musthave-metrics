// Package retry повторяет операции, завершившиеся временной ошибкой, с
// нарастающими паузами между попытками.
package retry

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// delays задаёт паузы перед повторными попытками: 1, 3 и 5 секунд.
var delays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// Do выполняет handler и повторяет его при временной ошибке.
//
// Функция isRetriable определяет, считается ли ошибка временной (см.
// [IsPGRetriable] и [IsHTTPRetriable]). Всего выполняется до четырёх
// попыток (первая плюс три повтора с паузами из delays). Ожидание паузы
// прерывается при отмене ctx — тогда возвращается ошибка контекста.
// Возвращается результат последней попытки.
func Do(ctx context.Context, isRetriable func(error) bool, handler func() error) error {
	err := handler()
	if err == nil || !isRetriable(err) {
		return err
	}

	for _, delay := range delays {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		err = handler()
		if err == nil || !isRetriable(err) {
			return err
		}
	}

	return err
}

// IsPGRetriable сообщает, является ли ошибка временной ошибкой PostgreSQL,
// которую имеет смысл повторить: сбой сериализации, взаимоблокировка,
// остановка/перегрузка сервера, исключения соединения, а также сетевые
// ошибки.
func IsPGRetriable(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case pgerrcode.SerializationFailure,
			pgerrcode.DeadlockDetected,
			pgerrcode.AdminShutdown,
			pgerrcode.TooManyConnections:
			return true
		}
		return pgerrcode.IsConnectionException(pgErr.Code)
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// IsHTTPRetriable сообщает, является ли ошибка временной сетевой ошибкой
// HTTP-запроса, которую имеет смысл повторить.
func IsHTTPRetriable(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}
