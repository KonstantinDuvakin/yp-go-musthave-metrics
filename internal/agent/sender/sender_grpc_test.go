package sender

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/tlstest"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	pb "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// fakeMetricsClient — подмена pb.MetricsClient, запоминающая вызовы.
//
// errs задаёт ошибки, которые вернут вызовы по порядку; когда список
// заканчивается, вызовы завершаются успешно. Если hang > 0, первые hang
// вызовов «зависают» до отмены своего контекста, как неотвечающий сервер.
type fakeMetricsClient struct {
	mu    sync.Mutex
	errs  []error
	hang  int
	reqs  []*pb.UpdateMetricsRequest
	metas []metadata.MD
	// left — сколько оставалось до дедлайна контекста в момент вызова
	// (0, если дедлайна нет).
	left []time.Duration
}

func (f *fakeMetricsClient) UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsRequest, _ ...grpc.CallOption) (*pb.UpdateMetricsResponse, error) {
	f.mu.Lock()

	md, _ := metadata.FromOutgoingContext(ctx)
	f.reqs = append(f.reqs, in)
	f.metas = append(f.metas, md)

	var left time.Duration
	if deadline, ok := ctx.Deadline(); ok {
		left = time.Until(deadline)
	}
	f.left = append(f.left, left)

	if f.hang > 0 {
		f.hang--
		f.mu.Unlock()

		// Так же ведёт себя настоящий клиент: ошибка контекста становится
		// кодом DeadlineExceeded или Canceled.
		<-ctx.Done()
		return nil, status.FromContextError(ctx.Err()).Err()
	}
	defer f.mu.Unlock()

	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		return nil, err
	}

	return &pb.UpdateMetricsResponse{}, nil
}

func (f *fakeMetricsClient) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.reqs)
}

// newGRPCSender создаёт Sender в gRPC-режиме поверх фейкового клиента.
func newGRPCSender(c pb.MetricsClient, ip net.IP) *Sender {
	return &Sender{
		grpcClient: c,
		grpcIP:     ip,
		transport:  GRPC,
	}
}

func TestNewSender_Transport(t *testing.T) {
	t.Run("empty grpc address selects HTTP", func(t *testing.T) {
		s, err := NewSender("127.0.0.1:8080", "", "")
		if err != nil {
			t.Fatalf("NewSender() вернул ошибку: %v", err)
		}

		if s.transport != HTTP {
			t.Errorf("transport = %q, want %q", s.transport, HTTP)
		}
		if s.grpcConn != nil || s.grpcClient != nil {
			t.Error("в HTTP-режиме gRPC-клиент создаваться не должен")
		}
		if err = s.Close(); err != nil {
			t.Errorf("Close() вернул ошибку: %v", err)
		}
	})

	t.Run("grpc address selects GRPC", func(t *testing.T) {
		// Соединение ленивое: сервер на этом адресе не нужен.
		certFile, _ := tlstest.WriteCert(t)
		s, err := NewSender("127.0.0.1:8080", "127.0.0.1:3200", certFile)
		if err != nil {
			t.Fatalf("NewSender() вернул ошибку: %v", err)
		}

		if s.transport != GRPC {
			t.Errorf("transport = %q, want %q", s.transport, GRPC)
		}
		if s.grpcConn == nil || s.grpcClient == nil {
			t.Fatal("в gRPC-режиме должны быть созданы соединение и клиент")
		}
		if !s.grpcIP.Equal(net.ParseIP("127.0.0.1")) {
			t.Errorf("grpcIP = %v, want 127.0.0.1", s.grpcIP)
		}
		if err = s.Close(); err != nil {
			t.Errorf("Close() вернул ошибку: %v", err)
		}
	})
}

func TestSendMetricsBatch_GRPC(t *testing.T) {
	gv := 42.5
	dv := int64(7)
	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &gv},
		{ID: "PollCount", MType: models.Counter, Delta: &dv},
	}

	fake := &fakeMetricsClient{}
	s := newGRPCSender(fake, net.ParseIP("10.0.0.5"))

	if err := s.SendMetricsBatch(context.Background(), batch, "ignored-key", nil); err != nil {
		t.Fatalf("SendMetricsBatch() вернул ошибку: %v", err)
	}

	if fake.calls() != 1 {
		t.Fatalf("UpdateMetrics вызван %d раз, want 1", fake.calls())
	}

	got := fake.reqs[0].GetMetrics()
	if len(got) != 2 {
		t.Fatalf("в запросе %d метрик, want 2", len(got))
	}

	if got[0].GetId() != "Alloc" || got[0].GetType() != pb.Metric_GAUGE || got[0].GetValue() != gv || got[0].GetDelta() != 0 {
		t.Errorf("gauge передан неверно: %v", got[0])
	}
	if got[1].GetId() != "PollCount" || got[1].GetType() != pb.Metric_COUNTER || got[1].GetDelta() != dv || got[1].GetValue() != 0 {
		t.Errorf("counter передан неверно: %v", got[1])
	}

	if ip := fake.metas[0].Get("x-real-ip"); len(ip) != 1 || ip[0] != "10.0.0.5" {
		t.Errorf("x-real-ip = %v, want [10.0.0.5]", ip)
	}
}

func TestSendMetricsBatch_GRPCWithoutIP(t *testing.T) {
	gv := 1.0
	batch := []models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &gv}}

	fake := &fakeMetricsClient{}
	s := newGRPCSender(fake, nil)

	if err := s.SendMetricsBatch(context.Background(), batch, "", nil); err != nil {
		t.Fatalf("SendMetricsBatch() вернул ошибку: %v", err)
	}

	if ip := fake.metas[0].Get("x-real-ip"); len(ip) != 0 {
		t.Errorf("x-real-ip = %v, а IP неизвестен — метаданных быть не должно", ip)
	}
}

func TestSendMetricsBatch_GRPCErrors(t *testing.T) {
	gv := 1.0
	batch := []models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &gv}}

	t.Run("non-retriable error returned immediately", func(t *testing.T) {
		fake := &fakeMetricsClient{errs: []error{status.Error(codes.PermissionDenied, "IP is not allowed")}}
		s := newGRPCSender(fake, net.ParseIP("10.0.0.5"))

		err := s.SendMetricsBatch(context.Background(), batch, "", nil)

		if status.Code(err) != codes.PermissionDenied {
			t.Errorf("код ошибки = %v, want %v (err: %v)", status.Code(err), codes.PermissionDenied, err)
		}
		if fake.calls() != 1 {
			t.Errorf("UpdateMetrics вызван %d раз, want 1: PermissionDenied повторять нельзя", fake.calls())
		}
	})

	t.Run("unavailable is retried", func(t *testing.T) {
		// Первая пауза retry.Do — 1 секунда, поэтому подтест длится около секунды.
		fake := &fakeMetricsClient{errs: []error{status.Error(codes.Unavailable, "connection refused")}}
		s := newGRPCSender(fake, net.ParseIP("10.0.0.5"))

		if err := s.SendMetricsBatch(context.Background(), batch, "", nil); err != nil {
			t.Fatalf("SendMetricsBatch() вернул ошибку: %v", err)
		}
		if fake.calls() != 2 {
			t.Errorf("UpdateMetrics вызван %d раз, want 2", fake.calls())
		}
	})
}

func TestSendMetricsBatch_GRPCDeadlinePerAttempt(t *testing.T) {
	gv := 1.0
	batch := []models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &gv}}

	// Первая попытка падает с Unavailable, вторая проходит. Между ними retry.Do
	// ждёт 1 секунду.
	fake := &fakeMetricsClient{errs: []error{status.Error(codes.Unavailable, "down")}}
	s := newGRPCSender(fake, nil)

	if err := s.SendMetricsBatch(context.Background(), batch, "", nil); err != nil {
		t.Fatalf("SendMetricsBatch() вернул ошибку: %v", err)
	}
	if len(fake.left) != 2 {
		t.Fatalf("UpdateMetrics вызван %d раз, want 2", len(fake.left))
	}

	// Если бы дедлайн ставился один раз снаружи retry.Do, ко второй попытке
	// от него осталось бы меньше timeoutCall минус пауза.
	const slack = 500 * time.Millisecond
	for i, left := range fake.left {
		if left <= timeoutCall-slack || left > timeoutCall {
			t.Errorf("попытка %d: до дедлайна %v, want около %v (у каждой попытки свой дедлайн)", i+1, left, timeoutCall)
		}
	}
}

func TestSendMetricsBatch_GRPCHungCallTimesOut(t *testing.T) {
	if testing.Short() {
		t.Skip("ждёт истечения дедлайна попытки")
	}

	gv := 1.0
	batch := []models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &gv}}

	// Попытка зависает: её должен прервать дедлайн timeoutCall с кодом
	// DeadlineExceeded. Повтора нет: сервер мог успеть сохранить пакет, и
	// повтор задвоил бы counter.
	fake := &fakeMetricsClient{hang: 1}
	s := newGRPCSender(fake, net.ParseIP("10.0.0.5"))

	start := time.Now()
	err := s.SendMetricsBatch(context.Background(), batch, "", nil)
	elapsed := time.Since(start)

	if status.Code(err) != codes.DeadlineExceeded {
		t.Errorf("код ошибки = %v, want %v (err: %v)", status.Code(err), codes.DeadlineExceeded, err)
	}
	if fake.calls() != 1 {
		t.Errorf("UpdateMetrics вызван %d раз, want 1: после DeadlineExceeded повторять нельзя", fake.calls())
	}
	if elapsed < timeoutCall {
		t.Errorf("отправка заняла %v: зависшая попытка должна была ждать дедлайн %v", elapsed, timeoutCall)
	}
	// Дедлайн попытки строится от ctx, поэтому метаданные сохраняются.
	if ip := fake.metas[0].Get("x-real-ip"); len(ip) != 1 || ip[0] != "10.0.0.5" {
		t.Errorf("x-real-ip = %v, want [10.0.0.5]", ip)
	}
}

func TestSendMetricsBatch_GRPCCancelStopsHungCall(t *testing.T) {
	gv := 1.0
	batch := []models.Metrics{{ID: "Alloc", MType: models.Gauge, Value: &gv}}

	fake := &fakeMetricsClient{hang: 1}
	s := newGRPCSender(fake, nil)

	// Отмена внешнего контекста (как sendCtx при остановке агента) должна
	// прервать зависший вызов сразу, не дожидаясь дедлайна, и без повтора.
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	start := time.Now()
	err := s.SendMetricsBatch(ctx, batch, "", nil)

	if status.Code(err) != codes.Canceled {
		t.Errorf("код ошибки = %v, want %v (err: %v)", status.Code(err), codes.Canceled, err)
	}
	if fake.calls() != 1 {
		t.Errorf("UpdateMetrics вызван %d раз, want 1: Canceled повторять нельзя", fake.calls())
	}
	if elapsed := time.Since(start); elapsed >= timeoutCall {
		t.Errorf("отправка заняла %v: отмена ctx должна прерывать вызов раньше дедлайна", elapsed)
	}
}

func TestNewSender_GRPCCertificate(t *testing.T) {
	t.Run("missing certificate file", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing.pem")

		if _, err := NewSender("127.0.0.1:8080", "127.0.0.1:3200", missing); err == nil {
			t.Error("NewSender() без файла сертификата должен вернуть ошибку")
		}
	})

	t.Run("empty certificate path", func(t *testing.T) {
		if _, err := NewSender("127.0.0.1:8080", "127.0.0.1:3200", ""); err == nil {
			t.Error("NewSender() с gRPC-адресом, но без сертификата должен вернуть ошибку: insecure не используется")
		}
	})

	t.Run("file without PEM certificate", func(t *testing.T) {
		bad := filepath.Join(t.TempDir(), "bad.pem")
		if err := os.WriteFile(bad, []byte("not a certificate"), 0o644); err != nil {
			t.Fatal(err)
		}

		if _, err := NewSender("127.0.0.1:8080", "127.0.0.1:3200", bad); err == nil {
			t.Error("NewSender() с некорректным PEM должен вернуть ошибку")
		}
	})

	t.Run("http mode ignores certificate", func(t *testing.T) {
		s, err := NewSender("127.0.0.1:8080", "", "")
		if err != nil {
			t.Fatalf("NewSender() в HTTP-режиме не должен требовать сертификат: %v", err)
		}
		_ = s.Close()
	})
}
