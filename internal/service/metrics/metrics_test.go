package metrics

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	pb "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/proto"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// fakeStorage запоминает пакет, переданный в SaveMetricsBatch. Остальные
// методы storage.MetricsStorage в тестах не вызываются.
type fakeStorage struct {
	storage.MetricsStorage
	saved [][]models.Metrics
	err   error
}

func (f *fakeStorage) SaveMetricsBatch(_ context.Context, metrics []models.Metrics) error {
	f.saved = append(f.saved, metrics)
	return f.err
}

func mustParseCIDR(t *testing.T, s string) *net.IPNet {
	t.Helper()
	_, ipNet, err := net.ParseCIDR(s)
	require.NoError(t, err)
	return ipNet
}

func TestCheckIPInterceptor(t *testing.T) {
	trusted := mustParseCIDR(t, "192.168.1.0/24")

	tests := []struct {
		name     string
		ipNet    *net.IPNet
		md       metadata.MD // nil — метаданных в контексте нет совсем
		wantCode codes.Code
	}{
		{name: "nil subnet allows without metadata", ipNet: nil, md: nil, wantCode: codes.OK},
		{name: "no metadata", ipNet: trusted, md: nil, wantCode: codes.PermissionDenied},
		{name: "no x-real-ip", ipNet: trusted, md: metadata.Pairs("other", "1"), wantCode: codes.PermissionDenied},
		{name: "invalid ip", ipNet: trusted, md: metadata.Pairs("x-real-ip", "not-an-ip"), wantCode: codes.PermissionDenied},
		{name: "ip outside subnet", ipNet: trusted, md: metadata.Pairs("x-real-ip", "10.0.0.1"), wantCode: codes.PermissionDenied},
		{name: "ip inside subnet", ipNet: trusted, md: metadata.Pairs("x-real-ip", "192.168.1.42"), wantCode: codes.OK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.md != nil {
				ctx = metadata.NewIncomingContext(ctx, tt.md)
			}

			called := false
			handler := func(ctx context.Context, req any) (any, error) {
				called = true
				return "ok", nil
			}

			resp, err := checkIPInterceptor(tt.ipNet)(ctx, nil, &grpc.UnaryServerInfo{}, handler)

			assert.Equal(t, tt.wantCode, status.Code(err))
			assert.Equal(t, tt.wantCode == codes.OK, called, "handler должен вызываться только при успешной проверке")
			if tt.wantCode == codes.OK {
				assert.Equal(t, "ok", resp)
			}
		})
	}
}

func TestMetrics_UpdateMetrics(t *testing.T) {
	t.Run("saves batch", func(t *testing.T) {
		store := &fakeStorage{}
		m := NewMetrics(store)

		req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
			pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42.5}.Build(),
			pb.Metric_builder{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 7}.Build(),
		}}.Build()

		_, err := m.UpdateMetrics(context.Background(), req)
		require.NoError(t, err)

		require.Len(t, store.saved, 1)
		got := store.saved[0]
		require.Len(t, got, 2)

		assert.Equal(t, "Alloc", got[0].ID)
		assert.Equal(t, models.Gauge, got[0].MType)
		require.NotNil(t, got[0].Value)
		assert.Equal(t, 42.5, *got[0].Value)
		assert.Nil(t, got[0].Delta, "у gauge delta быть не должно")

		assert.Equal(t, "PollCount", got[1].ID)
		assert.Equal(t, models.Counter, got[1].MType)
		require.NotNil(t, got[1].Delta)
		assert.Equal(t, int64(7), *got[1].Delta)
		assert.Nil(t, got[1].Value, "у counter value быть не должно")
	})

	invalid := []struct {
		name   string
		metric *pb.Metric
	}{
		{name: "empty id", metric: pb.Metric_builder{Id: "", Type: pb.Metric_GAUGE, Value: 1}.Build()},
		{name: "unknown type", metric: pb.Metric_builder{Id: "X", Type: pb.Metric_MType(7)}.Build()},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStorage{}
			req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{tt.metric}}.Build()

			_, err := NewMetrics(store).UpdateMetrics(context.Background(), req)

			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Empty(t, store.saved, "некорректный пакет не должен попадать в хранилище")
		})
	}

	t.Run("storage error", func(t *testing.T) {
		store := &fakeStorage{err: errors.New("db is down")}
		req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
			pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1}.Build(),
		}}.Build()

		_, err := NewMetrics(store).UpdateMetrics(context.Background(), req)

		assert.Equal(t, codes.Internal, status.Code(err))
	})
}

// TestMetrics_OverGRPC проверяет связку целиком: клиент кладёт x-real-ip в
// исходящие метаданные, а interceptor на сервере видит его во входящих.
func TestMetrics_OverGRPC(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	store := &fakeStorage{}

	// Тот же конструктор, что использует Run, — тест проверяет боевую сборку сервера.
	srv := NewMetrics(store).createGRPCServer(mustParseCIDR(t, "192.168.1.0/24"))
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	client := pb.NewMetricsClient(conn)
	req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1}.Build(),
	}}.Build()

	t.Run("trusted ip", func(t *testing.T) {
		ctx := metadata.AppendToOutgoingContext(context.Background(), "x-real-ip", "192.168.1.10")

		_, err := client.UpdateMetrics(ctx, req)

		require.NoError(t, err)
	})

	t.Run("untrusted ip", func(t *testing.T) {
		ctx := metadata.AppendToOutgoingContext(context.Background(), "x-real-ip", "10.0.0.1")

		_, err := client.UpdateMetrics(ctx, req)

		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("no ip", func(t *testing.T) {
		_, err := client.UpdateMetrics(context.Background(), req)

		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	assert.Len(t, store.saved, 1, "в хранилище должен попасть только запрос из доверенной подсети")
}

// blockingStorage задерживает SaveMetricsBatch, пока не закроют release.
// В started сигнализирует, что запрос дошёл до хранилища.
type blockingStorage struct {
	storage.MetricsStorage
	started chan struct{}
	release chan struct{}
}

func (b *blockingStorage) SaveMetricsBatch(context.Context, []models.Metrics) error {
	close(b.started)
	<-b.release
	return nil
}

// freeAddr возвращает свободный локальный адрес для запуска сервера.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

// startRun запускает Run в горутине и возвращает канал с его результатом.
func startRun(ctx context.Context, m *Metrics, addr string) <-chan error {
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx, addr, nil) }()
	return done
}

// newClient создаёт клиент к серверу на addr.
func newClient(t *testing.T, addr string) pb.MetricsClient {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return pb.NewMetricsClient(conn)
}

func TestMetrics_Run(t *testing.T) {
	req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1}.Build(),
	}}.Build()

	t.Run("busy port", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = l.Close() })

		err = NewMetrics(&fakeStorage{}).Run(context.Background(), l.Addr().String(), nil)

		assert.Error(t, err)
	})

	t.Run("stops on ctx cancel", func(t *testing.T) {
		addr := freeAddr(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := startRun(ctx, NewMetrics(&fakeStorage{}), addr)

		// WaitForReady ждёт, пока сервер начнёт принимать соединения.
		callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer callCancel()
		_, err := newClient(t, addr).UpdateMetrics(callCtx, req, grpc.WaitForReady(true))
		require.NoError(t, err)

		cancel()

		select {
		case err = <-done:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Run не завершился после отмены ctx")
		}
	})

	t.Run("waits for in-flight request", func(t *testing.T) {
		addr := freeAddr(t)
		store := &blockingStorage{started: make(chan struct{}), release: make(chan struct{})}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := startRun(ctx, NewMetrics(store), addr)

		callErr := make(chan error, 1)
		go func() {
			callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer callCancel()
			_, err := newClient(t, addr).UpdateMetrics(callCtx, req, grpc.WaitForReady(true))
			callErr <- err
		}()

		select {
		case <-store.started:
		case <-time.After(5 * time.Second):
			t.Fatal("запрос не дошёл до хранилища")
		}

		cancel()

		select {
		case <-done:
			t.Fatal("Run вернулся, не дождавшись запроса, который ещё выполняется")
		case <-time.After(100 * time.Millisecond):
		}

		close(store.release)

		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Run не завершился после окончания запроса")
		}
		assert.NoError(t, <-callErr, "начатый до остановки запрос должен завершиться успешно")
	})
}
