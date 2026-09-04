package retry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

var errRetriable = errors.New("retriable")

func alwaysRetriable(err error) bool { return err != nil }
func neverRetriable(error) bool      { return false }

func setFastDelays(t *testing.T, count int) {
	t.Helper()
	orig := delays
	d := make([]time.Duration, count)
	for i := range d {
		d[i] = time.Millisecond
	}
	delays = d
	t.Cleanup(func() { delays = orig })
}

func TestDo_SuccessFirstTry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), alwaysRetriable, func() error {
		calls++
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 1, calls, "при успехе повторов быть не должно")
}

func TestDo_NonRetriableReturnsImmediately(t *testing.T) {
	fatal := errors.New("fatal")
	calls := 0
	err := Do(context.Background(), neverRetriable, func() error {
		calls++
		return fatal
	})

	require.ErrorIs(t, err, fatal)
	require.Equal(t, 1, calls, "non-retriable ошибка не должна повторяться")
}

func TestDo_RetriableThenSuccess(t *testing.T) {
	setFastDelays(t, 3)

	calls := 0
	err := Do(context.Background(), alwaysRetriable, func() error {
		calls++
		if calls < 3 {
			return errRetriable
		}
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 3, calls, "должно хватить двух повторов до успеха")
}

func TestDo_ExhaustsRetries(t *testing.T) {
	setFastDelays(t, 3)

	calls := 0
	err := Do(context.Background(), alwaysRetriable, func() error {
		calls++
		return errRetriable
	})

	require.ErrorIs(t, err, errRetriable, "после всех попыток возвращается последняя ошибка")
	require.Equal(t, 4, calls, "1 основная попытка + 3 повтора")
}

func TestDo_CtxCancelledDuringBackoff(t *testing.T) {
	orig := delays
	delays = []time.Duration{time.Hour, time.Hour, time.Hour}
	t.Cleanup(func() { delays = orig })

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем сразу

	calls := 0
	err := Do(ctx, alwaysRetriable, func() error {
		calls++
		return errRetriable
	})

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls, "первая попытка была, до повторов дело не дошло из-за отмены")
}

func TestIsPGRetriable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"class 08 — connection exception", &pgconn.PgError{Code: pgerrcode.ConnectionException}, true},
		{"class 08 — connection failure", &pgconn.PgError{Code: pgerrcode.ConnectionFailure}, true},
		{"unique violation (23505) — не retriable", &pgconn.PgError{Code: pgerrcode.UniqueViolation}, false},
		{"обёрнутый class 08 через %w", fmt.Errorf("exec metric: %w", &pgconn.PgError{Code: pgerrcode.ConnectionFailure}), true},
		{"обычная ошибка", errors.New("boom"), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsPGRetriable(tt.err))
		})
	}
}

func TestIsHttpRetriable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"обычная ошибка", errors.New("boom"), false},
		{"net.OpError (отказ соединения)", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, true},
		{"обёрнутая сетевая ошибка через %w", fmt.Errorf("post: %w", &net.OpError{Op: "dial", Err: errors.New("refused")}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsHTTPRetriable(tt.err))
		})
	}
}
