// Package metrics реализует gRPC-сервис Metrics: приём пакетов метрик от
// агента и их сохранение в хранилище сервера.
//
// Запросы проходят через interceptor, который пропускает только агентов из
// доверенной подсети (см. [Metrics.Run]).
package metrics

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	pb "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/proto"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Metrics — реализация gRPC-сервиса [pb.MetricsServer] поверх хранилища
// метрик. Создаётся через [NewMetrics].
type Metrics struct {
	pb.UnimplementedMetricsServer
	store storage.MetricsStorage
}

// NewMetrics создаёт [Metrics], сохраняющий метрики в store.
func NewMetrics(store storage.MetricsStorage) *Metrics {
	return &Metrics{
		store: store,
	}
}

// checkIPInterceptor возвращает unary-interceptor, который пропускает
// только запросы, чей IP из метаданных x-real-ip входит в ipNet.
//
// Если ipNet равен nil, проверка отключена и пропускаются все запросы. Если
// метаданных или ключа x-real-ip нет, а также если IP не входит в подсеть,
// запрос отклоняется с кодом [codes.PermissionDenied].
func checkIPInterceptor(ipNet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if ipNet != nil {
			md, ok := metadata.FromIncomingContext(ctx)

			if !ok {
				return nil, status.Error(codes.PermissionDenied, "IP is not provided in metadata")
			}

			values := md.Get("x-real-ip")
			if len(values) == 0 {
				return nil, status.Error(codes.PermissionDenied, "IP is not provided in metadata")
			}

			ip := net.ParseIP(values[0])

			ok = ipNet.Contains(ip)
			if !ok {
				return nil, status.Error(codes.PermissionDenied, "IP is not allowed")
			}
		}

		return handler(ctx, req)
	}
}

// Run запускает gRPC-сервер на addr с сервисом m и проверкой подсети ipNet
// (см. checkIPInterceptor; nil отключает проверку).
//
// Блокирует выполнение, пока не отменят ctx или сервер не завершится с
// ошибкой. При отмене ctx сервер останавливается штатно: новые запросы не
// принимаются, текущие дорабатывают до конца (см. grpc.Server.GracefulStop),
// после чего Run возвращает nil. Возвращает ошибку, если не удалось начать
// слушать addr или если сервер остановился сам.
func (m *Metrics) Run(ctx context.Context, addr string, ipNet *net.IPNet) error {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("ошибка при инициализации listener: %w", err)
	}
	defer listen.Close()

	s := m.createGRPCServer(ipNet)

	errServe := make(chan error, 1)
	go func() {
		errServe <- s.Serve(listen)
	}()

	select {
	case <-ctx.Done():
		t := time.AfterFunc(5*time.Second, s.Stop)
		defer t.Stop()

		s.GracefulStop()
		return nil
	case err = <-errServe:
		return fmt.Errorf("ошибка при работе сервера: %w", err)
	}
}

// createGRPCServer создаёт gRPC-сервер с сервисом m и interceptor'ом
// проверки подсети ipNet (см. checkIPInterceptor; nil отключает проверку).
//
// Сервер ещё не запущен: его нужно передать listener'у через Serve. Вынесен
// из [Metrics.Run], чтобы тесты собирали сервер тем же кодом.
func (m *Metrics) createGRPCServer(ipNet *net.IPNet) *grpc.Server {
	s := grpc.NewServer(grpc.UnaryInterceptor(checkIPInterceptor(ipNet)))

	pb.RegisterMetricsServer(s, m)

	return s
}

// UpdateMetrics сохраняет пакет метрик из in одним вызовом
// SaveMetricsBatch.
//
// Для counter сохраняется только delta, для gauge — только value. Если
// у какой-то метрики пустой id или неизвестный тип, пакет целиком
// отклоняется с кодом [codes.InvalidArgument]. Ошибка хранилища
// возвращается с кодом [codes.Internal].
func (m *Metrics) UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	var response pb.UpdateMetricsResponse
	metricsList := make([]models.Metrics, 0, len(in.GetMetrics()))

	for _, metricPb := range in.GetMetrics() {
		var metric models.Metrics
		mType := metricPb.GetType()

		if metric.ID = metricPb.GetId(); metric.ID == "" {
			logger.Log.Warn("Отсутствует ID метрики")
			return nil, status.Error(codes.InvalidArgument, "Отсутствует ID метрики")
		}

		switch mType {
		case pb.Metric_COUNTER:
			metric.Delta = new(metricPb.GetDelta())
			metric.MType = models.Counter
		case pb.Metric_GAUGE:
			metric.Value = new(metricPb.GetValue())
			metric.MType = models.Gauge
		default:
			logger.Log.Warn("Неизвестный тип метрики", zap.String("metric_type", mType.String()))
			return nil, status.Error(codes.InvalidArgument, "Неизвестный тип метрики")
		}

		metricsList = append(metricsList, metric)
	}

	err := m.store.SaveMetricsBatch(ctx, metricsList)
	if err != nil {
		logger.Log.Error("Не удалось сохранить метрики", zap.Error(err))
		return nil, status.Error(codes.Internal, "Не удалось сохранить метрики")
	}

	return &response, nil
}
