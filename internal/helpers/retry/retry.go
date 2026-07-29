package retry

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var delays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

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

func IsPGRetriable(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgerrcode.IsConnectionException(pgErr.Code)
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func IsHttpRetriable(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}
