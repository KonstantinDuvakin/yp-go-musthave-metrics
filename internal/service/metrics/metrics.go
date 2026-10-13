// Package metrics реализует gRPC-сервис Metrics: приём пакетов метрик от
// агента. Сохранение и запись аудита выполняет сервисный слой
// (updatemetrics), общий с HTTP-обработчиком; здесь только преобразование
// protobuf-типов и проверка подсети.
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
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// batchUpdater — то, что нужно gRPC-сервису от сервисного слоя: принять
// пакет метрик от отправителя ip. Реализуется updatemetrics.Service.
type batchUpdater interface {
	UpdateBatch(ctx context.Context, metrics []models.Metrics, ip string) error
}

// Metrics — реализация gRPC-сервиса [pb.MetricsServer] поверх сервисного
// слоя приёма метрик. Создаётся через [NewMetrics].
type Metrics struct {
	pb.UnimplementedMetricsServer
	batchService batchUpdater
}

// NewMetrics создаёт [Metrics], передающий принятые пакеты в
// batchService.UpdateBatch. batchService не должен быть nil.
func NewMetrics(batchService batchUpdater) *Metrics {
	return &Metrics{
		batchService: batchService,
	}
}

// checkIPInterceptor возвращает unary-interceptor, который пропускает
// только запросы, чей IP из метаданных x-real-ip входит в ipNet.
//
// Если ipNet равен nil, проверка отключена и пропускаются все запросы. Если
// метаданных или ключа x-real-ip нет, а также если IP не входит в подсеть,
// запрос отклоняется с кодом [codes.PermissionDenied].
//
// Interceptor только проверяет доступ и контекст не меняет. IP для аудита
// обработчик получает сам, через clientIP.
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

			ipRaw := net.ParseIP(values[0])

			ok = ipNet.Contains(ipRaw)
			if !ok {
				return nil, status.Error(codes.PermissionDenied, "IP is not allowed")
			}
		}

		return handler(ctx, req)
	}
}

// Run запускает gRPC-сервер на addr с сервисом m и проверкой подсети ipNet
// (см. checkIPInterceptor; nil отключает проверку). Сервер принимает только
// TLS-соединения с сертификатом certFile и ключом keyFile (см.
// createGRPCServer).
//
// Блокирует выполнение, пока не отменят ctx или сервер не завершится с
// ошибкой. При отмене ctx сервер останавливается штатно: новые запросы не
// принимаются, текущие дорабатывают до конца (см. grpc.Server.GracefulStop),
// после чего Run возвращает nil. Возвращает ошибку, если не удалось начать
// слушать addr, загрузить сертификат или ключ, или если сервер остановился
// сам.
func (m *Metrics) Run(ctx context.Context, addr, certFile, keyFile string, ipNet *net.IPNet) error {
	listen, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("ошибка при инициализации listener: %w", err)
	}
	defer listen.Close()

	s, err := m.createGRPCServer(ipNet, certFile, keyFile)
	if err != nil {
		return err
	}

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
// Транспорт — TLS с сертификатом certFile и приватным ключом keyFile в PEM
// (см. cmd/certgen); соединения без TLS сервер не принимает. Возвращает
// ошибку, если сертификат или ключ не загружаются.
//
// Сервер ещё не запущен: его нужно передать listener'у через Serve. Вынесен
// из [Metrics.Run], чтобы тесты собирали сервер тем же кодом.
func (m *Metrics) createGRPCServer(ipNet *net.IPNet, certFile, keyFile string) (*grpc.Server, error) {
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't construct TLS credentials: %w", err)
	}

	s := grpc.NewServer(grpc.Creds(creds), grpc.UnaryInterceptor(checkIPInterceptor(ipNet)))

	pb.RegisterMetricsServer(s, m)

	return s, nil
}

// UpdateMetrics переводит пакет метрик из in в []models.Metrics и передаёт
// его в UpdateBatch сервисного слоя вместе с IP отправителя. Сервис
// сохраняет пакет и пишет событие аудита.
//
// IP для аудита определяет clientIP: x-real-ip из метаданных, а если его нет
// или он некорректен, то адрес соединения без порта. От проверки подсети
// это не зависит.
//
// Для counter передаётся только delta, для gauge — только value. Если
// у какой-то метрики пустой id или неизвестный тип, пакет целиком
// отклоняется с кодом [codes.InvalidArgument], и сервис не вызывается.
// Ошибка сервиса возвращается с кодом [codes.Internal].
func (m *Metrics) UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	var response pb.UpdateMetricsResponse
	metricsList := make([]models.Metrics, 0, len(in.GetMetrics()))

	for _, metricPb := range in.GetMetrics() {
		var metric models.Metrics
		mType := metricPb.GetType()

		if metric.ID = metricPb.GetId(); metric.ID == "" {
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
			return nil, status.Error(codes.InvalidArgument, "Неизвестный тип метрики")
		}

		metricsList = append(metricsList, metric)
	}

	ip := clientIP(ctx)

	err := m.batchService.UpdateBatch(ctx, metricsList, ip)
	if err != nil {
		logger.Log.Error("Не удалось сохранить метрики", zap.Error(err))
		return nil, status.Error(codes.Internal, "Не удалось сохранить метрики")
	}

	return &response, nil
}

// clientIP возвращает IP отправителя для аудита: x-real-ip из метаданных
// (в нормализованной записи), а если его нет или он некорректен — адрес
// соединения без порта. Адрес без порта возвращается как есть.
//
// Функция работает по принципу «лучшее, что удалось узнать»: не возвращает
// ошибку и ничего не логирует, потому что отсутствие x-real-ip — штатный
// случай. Если корректного x-real-ip нет и в контексте нет peer, возвращает
// пустую строку.
func clientIP(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get("x-real-ip"); len(v) > 0 {
			if ip := net.ParseIP(v[0]); ip != nil {
				return ip.String()
			}
		}
	}

	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}

	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}
	return host
}
