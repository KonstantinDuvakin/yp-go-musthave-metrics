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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// delays задаёт паузы перед повторными попытками: 1, 3 и 5 секунд.
var delays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// Do выполняет handler и повторяет его при временной ошибке.
//
// Функция isRetriable определяет, считается ли ошибка временной (см.
// [IsPGRetriable], [IsHTTPRetriable] и [IsGRPCRetriable]). Всего выполняется до четырёх
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

// IsGRPCRetriable сообщает, является ли ошибка gRPC-вызова временной, то
// есть имеет ли смысл повторить вызов. Ошибка может быть обёрнута через %w.
//
// Временными считаются ошибки с кодами:
//   - [codes.Unavailable] — сервер недоступен или соединение разорвано;
//   - [codes.DeadlineExceeded] — истёк дедлайн вызова.
//
// При DeadlineExceeded сервер мог успеть выполнить запрос и не успеть
// ответить, поэтому повтор неидемпотентного вызова (например, с дельтой
// counter) может применить его дважды. Остальные коды, в том числе
// [codes.Canceled], временными не считаются.
func IsGRPCRetriable(err error) bool {
	if err == nil {
		return false
	}
	isUnavailable := status.Code(err) == codes.Unavailable
	isDeadLineExceeded := status.Code(err) == codes.DeadlineExceeded
	return isUnavailable || isDeadLineExceeded
}
