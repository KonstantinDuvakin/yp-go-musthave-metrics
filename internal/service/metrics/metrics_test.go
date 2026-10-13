package metrics

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/tlstest"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	pb "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/proto"
	updatemetrics "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/update_metrics"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
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

// auditRecorder — AuditSender, запоминающий события аудита. Вызывается из горутин
// gRPC-сервера, поэтому защищён мьютексом.
type auditRecorder struct {
	mu     sync.Mutex
	events []models.Audit
}

func (a *auditRecorder) SendEvent(event models.Audit) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, event)
}

func (a *auditRecorder) all() []models.Audit {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]models.Audit(nil), a.events...)
}

// noAudit — заглушка AuditSender для тестов, которым аудит не важен.
type noAudit struct{}

func (noAudit) SendEvent(models.Audit) {}

// newMetrics собирает Metrics поверх настоящего сервиса updatemetrics, как в main.
func newMetrics(store storage.MetricsStorage, audit updatemetrics.AuditSender) *Metrics {
	return NewMetrics(updatemetrics.New(store, audit))
}

// peerIP — адрес соединения, который peerCtx кладёт в контекст.
const peerIP = "10.1.2.3"

// peerCtx возвращает контекст с адресом соединения, как его заполняет
// настоящий gRPC-сервер: вызовы UpdateMetrics в обход сервера должны его иметь.
func peerCtx() context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP(peerIP), Port: 51234},
	})
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
		audit := &auditRecorder{}
		m := newMetrics(store, audit)

		req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
			pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 42.5}.Build(),
			pb.Metric_builder{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 7}.Build(),
		}}.Build()

		_, err := m.UpdateMetrics(peerCtx(), req)
		require.NoError(t, err)

		events := audit.all()
		require.Len(t, events, 1, "после сохранения пакета должно уйти одно событие аудита")
		assert.Equal(t, []string{"Alloc", "PollCount"}, events[0].Metrics)
		assert.NotZero(t, events[0].TS)
		// x-real-ip в запросе нет — clientIP берёт адрес соединения без порта.
		assert.Equal(t, peerIP, events[0].IPAddress)

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
			audit := &auditRecorder{}
			req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{tt.metric}}.Build()

			_, err := newMetrics(store, audit).UpdateMetrics(peerCtx(), req)

			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Empty(t, store.saved, "некорректный пакет не должен попадать в хранилище")
			assert.Empty(t, audit.all(), "некорректный пакет не должен попадать в аудит")
		})
	}

	t.Run("storage error", func(t *testing.T) {
		store := &fakeStorage{err: errors.New("db is down")}
		req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
			pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1}.Build(),
		}}.Build()

		audit := &auditRecorder{}

		_, err := newMetrics(store, audit).UpdateMetrics(peerCtx(), req)

		assert.Equal(t, codes.Internal, status.Code(err))
		assert.Empty(t, audit.all(), "несохранённый пакет не должен попадать в аудит")
	})
}

// TestMetrics_OverGRPC проверяет связку целиком: клиент кладёт x-real-ip в
// исходящие метаданные, а interceptor на сервере видит его во входящих.
func TestMetrics_OverGRPC(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	store := &fakeStorage{}
	audit := &auditRecorder{}

	// Тот же конструктор, что использует Run, — тест проверяет боевую сборку сервера.
	certFile, keyFile := tlstest.WriteCert(t)
	srv, err := newMetrics(store, audit).createGRPCServer(mustParseCIDR(t, "192.168.1.0/24"), certFile, keyFile)
	require.NoError(t, err)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		// Адрес bufnet не совпадает с SAN сертификата, поэтому имя для проверки задаём явно.
		grpc.WithTransportCredentials(clientCreds(t, certFile, "localhost")),
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

	// gRPC-путь пишет аудит так же, как HTTP; IP берётся из x-real-ip.
	events := audit.all()
	require.Len(t, events, 1, "в аудит должен попасть только запрос из доверенной подсети")
	assert.Equal(t, "192.168.1.10", events[0].IPAddress)
	assert.Equal(t, []string{"Alloc"}, events[0].Metrics)
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
func startRun(ctx context.Context, m *Metrics, addr, certFile, keyFile string) <-chan error {
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx, addr, certFile, keyFile, nil) }()
	return done
}

// newClient создаёт TLS-клиент к серверу на addr, доверяющий сертификату certFile.
func newClient(t *testing.T, addr, certFile string) pb.MetricsClient {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(clientCreds(t, certFile, "")))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return pb.NewMetricsClient(conn)
}

func TestMetrics_Run(t *testing.T) {
	certFile, keyFile := tlstest.WriteCert(t)

	req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1}.Build(),
	}}.Build()

	t.Run("busy port", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = l.Close() })

		err = newMetrics(&fakeStorage{}, noAudit{}).Run(context.Background(), l.Addr().String(), certFile, keyFile, nil)

		assert.Error(t, err)
	})

	t.Run("stops on ctx cancel", func(t *testing.T) {
		addr := freeAddr(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := startRun(ctx, newMetrics(&fakeStorage{}, noAudit{}), addr, certFile, keyFile)

		// WaitForReady ждёт, пока сервер начнёт принимать соединения.
		callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer callCancel()
		_, err := newClient(t, addr, certFile).UpdateMetrics(callCtx, req, grpc.WaitForReady(true))
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
		done := startRun(ctx, newMetrics(store, noAudit{}), addr, certFile, keyFile)

		callErr := make(chan error, 1)
		go func() {
			callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer callCancel()
			_, err := newClient(t, addr, certFile).UpdateMetrics(callCtx, req, grpc.WaitForReady(true))
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

// TestMetrics_AuditIPFromConnection проверяет на настоящем TCP-порту, что без
// доверенной подсети и без x-real-ip в аудит попадает адрес соединения без
// порта. Через bufconn это не проверить: там адрес соединения не IP.
func TestMetrics_AuditIPFromConnection(t *testing.T) {
	certFile, keyFile := tlstest.WriteCert(t)
	addr := freeAddr(t)
	audit := &auditRecorder{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := startRun(ctx, newMetrics(&fakeStorage{}, audit), addr, certFile, keyFile)

	req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1}.Build(),
	}}.Build()

	callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer callCancel()
	_, err := newClient(t, addr, certFile).UpdateMetrics(callCtx, req, grpc.WaitForReady(true))
	require.NoError(t, err)

	events := audit.all()
	require.Len(t, events, 1)
	assert.Equal(t, "127.0.0.1", events[0].IPAddress)

	cancel()
	require.NoError(t, <-done)
}

func TestClientIP(t *testing.T) {
	withPeer := func(ctx context.Context, addr net.Addr) context.Context {
		return peer.NewContext(ctx, &peer.Peer{Addr: addr})
	}
	tcp := &net.TCPAddr{IP: net.ParseIP(peerIP), Port: 51234}

	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "x-real-ip wins over connection address",
			ctx:  withPeer(metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "192.168.1.42")), tcp),
			want: "192.168.1.42",
		},
		{
			name: "x-real-ip is normalized",
			ctx:  withPeer(metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "2001:DB8:0:0::1")), tcp),
			want: "2001:db8::1",
		},
		{
			name: "no x-real-ip — connection address without port",
			ctx:  withPeer(metadata.NewIncomingContext(context.Background(), metadata.Pairs("other", "1")), tcp),
			want: peerIP,
		},
		{
			name: "invalid x-real-ip — connection address",
			ctx:  withPeer(metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-real-ip", "not-an-ip")), tcp),
			want: peerIP,
		},
		{
			name: "no metadata — connection address",
			ctx:  withPeer(context.Background(), tcp),
			want: peerIP,
		},
		{
			name: "address without port is used as is",
			ctx:  withPeer(context.Background(), &net.UnixAddr{Name: "bufnet", Net: "unix"}),
			want: "bufnet",
		},
		{
			name: "no metadata and no peer — empty, no panic",
			ctx:  context.Background(),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, clientIP(tt.ctx))
		})
	}
}

// clientCreds возвращает TLS-учётные данные клиента, доверяющего сертификату
// certFile. serverName задаёт имя для проверки сертификата, если адрес
// подключения с ним не совпадает (например, bufnet); пустая строка — брать
// имя из адреса.
func clientCreds(t *testing.T, certFile, serverName string) credentials.TransportCredentials {
	t.Helper()
	creds, err := credentials.NewClientTLSFromFile(certFile, serverName)
	require.NoError(t, err)
	return creds
}

// TestMetrics_RequiresTLS проверяет, что сервер принимает только TLS-клиентов,
// доверяющих его сертификату (замечание ревьюера: без insecure).
func TestMetrics_RequiresTLS(t *testing.T) {
	certFile, keyFile := tlstest.WriteCert(t)
	addr := freeAddr(t)
	store := &fakeStorage{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := startRun(ctx, newMetrics(store, noAudit{}), addr, certFile, keyFile)

	req := pb.UpdateMetricsRequest_builder{Metrics: []*pb.Metric{
		pb.Metric_builder{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1}.Build(),
	}}.Build()

	call := func(t *testing.T, creds credentials.TransportCredentials) error {
		t.Helper()
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })

		callCtx, callCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer callCancel()
		_, err = pb.NewMetricsClient(conn).UpdateMetrics(callCtx, req)
		return err
	}

	// Ждём, пока сервер начнёт принимать соединения.
	readyCtx, readyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readyCancel()
	_, err := newClient(t, addr, certFile).UpdateMetrics(readyCtx, req, grpc.WaitForReady(true))
	require.NoError(t, err, "клиент с доверенным сертификатом должен проходить")

	t.Run("plaintext client is rejected", func(t *testing.T) {
		err := call(t, insecure.NewCredentials())
		assert.Equal(t, codes.Unavailable, status.Code(err), "без TLS соединение не устанавливается: %v", err)
	})

	t.Run("client trusting another certificate is rejected", func(t *testing.T) {
		otherCert, _ := tlstest.WriteCert(t)
		err := call(t, clientCreds(t, otherCert, ""))
		assert.Equal(t, codes.Unavailable, status.Code(err), "чужой сертификат не проходит проверку: %v", err)
	})

	assert.Len(t, store.saved, 1, "в хранилище попадает только запрос по доверенному TLS")

	cancel()
	require.NoError(t, <-done)
}

func TestMetrics_Run_MissingCertificate(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.pem")

	err := newMetrics(&fakeStorage{}, noAudit{}).Run(context.Background(), freeAddr(t), missing, missing, nil)

	assert.Error(t, err, "без сертификата сервер не должен стартовать")
}
