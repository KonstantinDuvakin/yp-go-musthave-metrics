// Package sender предназначен для отправки метрик на сервер по HTTP или gRPC.
//
// Пакет инкапсулирует resty-клиент и gRPC-клиент сервиса Metrics и
// предоставляет методы для отправки метрик как по URL-параметрам, так и в
// формате JSON (с gzip-сжатием, подписью тела запроса и повторными
// попытками при временных ошибках). Пакеты метрик уходят через тот
// транспорт, который выбран при создании [Sender] (см. [NewSender]).
package sender

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/crypto"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/hash"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/retry"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	pb "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/proto"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// Transport — протокол, по которому [Sender.SendMetricsBatch] отправляет
// пакеты метрик на сервер.
type Transport string

const (
	// HTTP — отправка пакетов POST-запросом на эндпоинт /updates.
	HTTP Transport = "HTTP"
	// GRPC — отправка пакетов вызовом Metrics.UpdateMetrics.
	GRPC Transport = "GRPC"
)

// timeoutCall — сколько агент ждёт сервер в одном обращении: таймаут
// HTTP-запроса, дедлайн одной попытки gRPC-вызова и минимальное время на
// установку gRPC-соединения. Паузы между повторами (см. [retry.Do]) в него
// не входят.
const timeoutCall = 2 * time.Second

// Sender отправляет метрики на сервер сбора метрик.
//
// Хранит настроенный HTTP-клиент [resty.Client] с заданными базовым URL,
// заголовками и таймаутом, а в gRPC-режиме ещё и соединение с
// gRPC-сервером. Создаётся через [NewSender]; после использования
// соединение нужно закрыть через [Sender.Close].
type Sender struct {
	client     *resty.Client
	grpcClient pb.MetricsClient
	grpcConn   *grpc.ClientConn
	grpcIP     net.IP
	transport  Transport
}

// NewSender создаёт [Sender] и выбирает транспорт для пакетов метрик.
//
// baseURL — адрес HTTP-сервера без схемы (например, "localhost:8080");
// схема http:// добавляется автоматически. Клиенту задаётся таймаут
// timeoutCall и заголовок Content-Type: text/plain по умолчанию.
//
// grpcAddr — адрес gRPC-сервера в формате host:port. Если он пуст, пакеты
// отправляются по HTTP ([HTTP]), иначе по gRPC ([GRPC]).
//
// certFile — путь к PEM-сертификату gRPC-сервера. Соединение с сервером
// идёт по TLS, и агент доверяет только этому сертификату (самоподписанному,
// см. cmd/certgen). Хост из grpcAddr должен быть в SAN сертификата. В
// HTTP-режиме certFile не используется.
//
// Агент передаёт свой IP-адрес, чтобы сервер сверил его с доверенной
// подсетью. Адрес определяется по маршруту до сервера выбранного
// транспорта (см. getLocalIP): в HTTP-режиме он уходит в заголовке
// X-Real-IP всех запросов, в gRPC-режиме — в метаданных x-real-ip. Если
// адрес определить не удалось, пишется предупреждение в лог, и IP не
// передаётся.
//
// Возвращает ошибку, если не удалось создать gRPC-клиент, в том числе если
// в gRPC-режиме certFile не задан, не читается или не содержит
// PEM-сертификата.
func NewSender(baseURL, grpcAddr, certFile string) (*Sender, error) {
	client := resty.New()
	client.SetBaseURL("http://" + baseURL)
	client.SetHeader("Content-Type", "text/plain")
	client.SetTimeout(timeoutCall)

	sender := &Sender{
		client: client,
	}

	var ip net.IP
	var err error

	if grpcAddr != "" {
		ip, err = getLocalIP(grpcAddr)
		if err != nil {
			logger.Log.Warn("Couldn't get local IP for grpc", zap.Error(err))
		}

		c, conn, grpcErr := createGRPCClient(grpcAddr, certFile)
		if grpcErr != nil {
			return nil, grpcErr
		}

		sender.transport = GRPC
		sender.grpcClient = c
		sender.grpcConn = conn
		sender.grpcIP = ip
	} else {
		ip, err = getLocalIP(baseURL)
		if err != nil {
			logger.Log.Warn("Couldn't get local IP for http", zap.Error(err))
		} else {
			client.SetHeader("X-Real-IP", ip.String())
		}

		sender.transport = HTTP
	}

	return sender, nil
}

// SendMetrics отправляет метрику POST-запросом по URL с параметрами в пути.
//
// url — относительный путь вида /update/{type}/{name}/{value}, который
// удобно собирать с помощью [URLBuilder]. Выполнение прерывается при
// отмене ctx.
func (s *Sender) SendMetrics(ctx context.Context, url string) error {
	_, err := s.client.R().SetContext(ctx).Post(url)
	return err
}

// SendMetricsJSON отправляет одну метрику в формате JSON на эндпоинт /update.
//
// Тело запроса сериализуется в JSON и сжимается gzip. Если hashKey не пуст,
// к запросу добавляется заголовок HashSHA256 с подписью исходных данных.
// Запрос повторяется при временных сетевых и HTTP-ошибках согласно
// [retry.Do] и [retry.IsHTTPRetriable].
func (s *Sender) SendMetricsJSON(ctx context.Context, body models.Metrics, hashKey string) error {
	bufGZip := bytes.NewBuffer(nil)
	zw := gzip.NewWriter(bufGZip)

	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	var sign string
	if hashKey != "" {
		sign = hash.CreateHeaderHash(data, hashKey)
	}

	_, err = zw.Write(data)
	if err != nil {
		return err
	}

	if err = zw.Close(); err != nil {
		return err
	}

	return retry.Do(ctx, retry.IsHTTPRetriable, func() error {
		req := s.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(bufGZip.Bytes())

		if hashKey != "" {
			req.SetHeader("HashSHA256", sign)
		}

		resp, err := req.Post("/update")

		if err != nil {
			return err
		}

		if resp.IsError() {
			return fmt.Errorf("batch upload failed: status %d", resp.StatusCode())
		}

		return nil
	})
}

// SendMetricsBatch отправляет пакет метрик одним запросом через транспорт,
// выбранный в [NewSender].
//
// В HTTP-режиме пакет уходит в формате JSON на эндпоинт /updates: тело
// сжимается gzip, при непустом hashKey добавляется подпись HashSHA256, при
// непустом pubKey тело шифруется. Возвращает ошибку, если сервер ответил
// статусом >= 400.
//
// В gRPC-режиме пакет уходит вызовом Metrics.UpdateMetrics; hashKey и
// pubKey при этом не используются.
//
// В обоих режимах каждая попытка ограничена timeoutCall, а запрос
// повторяется при временных ошибках (см. [retry.Do]). Отмена ctx прерывает
// и текущую попытку, и паузу перед следующей.
func (s *Sender) SendMetricsBatch(ctx context.Context, metrics []models.Metrics, hashKey string, pubKey *rsa.PublicKey) error {
	if s.transport == GRPC {
		return s.sendBatchGRPC(ctx, metrics)
	}
	return s.sendBatchHTTP(ctx, metrics, hashKey, pubKey)
}

// Close закрывает соединение с gRPC-сервером. В HTTP-режиме ничего не
// делает и возвращает nil.
func (s *Sender) Close() error {
	if s.grpcConn != nil {
		return s.grpcConn.Close()
	}
	return nil
}

// sendBatchHTTP отправляет пакет метрик на эндпоинт /updates (см.
// [Sender.SendMetricsBatch]). Повторяет запрос при временных сетевых
// ошибках согласно [retry.IsHTTPRetriable].
func (s *Sender) sendBatchHTTP(ctx context.Context, metrics []models.Metrics, hashKey string, pubKey *rsa.PublicKey) error {
	var body []byte
	bufGZip := bytes.NewBuffer(nil)
	zw := gzip.NewWriter(bufGZip)

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	var sign string
	if hashKey != "" {
		sign = hash.CreateHeaderHash(data, hashKey)
	}

	_, err = zw.Write(data)
	if err != nil {
		return err
	}

	if err = zw.Close(); err != nil {
		return err
	}

	body = bufGZip.Bytes()

	if pubKey != nil {
		body, err = crypto.EncryptBodyWithKey(body, pubKey)
		if err != nil {
			return err
		}
	}

	return retry.Do(ctx, retry.IsHTTPRetriable, func() error {
		req := s.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(body)

		if hashKey != "" {
			req.SetHeader("HashSHA256", sign)
		}

		if pubKey != nil {
			req.SetHeader("X-Encrypted", "true")
		}

		resp, err := req.Post("/updates")

		if err != nil {
			return err
		}

		if resp.IsError() {
			return fmt.Errorf("batch upload failed: status %d", resp.StatusCode())
		}

		return nil
	})
}

// sendBatchGRPC переводит метрики в [pb.Metric] и отправляет их одним
// вызовом UpdateMetrics. Если IP агента известен, он добавляется в
// исходящие метаданные под ключом x-real-ip.
//
// У каждой попытки свой дедлайн timeoutCall, производный от ctx: паузы
// между повторами его не расходуют, а отмена ctx прерывает вызов с кодом
// Canceled. Повторяет вызов только при временных ошибках согласно
// [retry.IsGRPCRetriable]. Если истёк дедлайн попытки, вызов не повторяется
// и возвращается ошибка с кодом DeadlineExceeded: сервер мог успеть сохранить
// пакет, и повтор задвоил бы значения counter.
func (s *Sender) sendBatchGRPC(ctx context.Context, metrics []models.Metrics) error {
	metricsProto := make([]*pb.Metric, 0, len(metrics))

	for _, m := range metrics {
		var mType pb.Metric_MType
		var value float64
		var delta int64

		if m.MType == models.Gauge {
			value = *m.Value
			mType = pb.Metric_GAUGE
		} else {
			delta = *m.Delta
			mType = pb.Metric_COUNTER
		}

		mProto := pb.Metric_builder{Id: m.ID, Delta: delta, Value: value, Type: mType}.Build()

		metricsProto = append(metricsProto, mProto)
	}

	if s.grpcIP != nil {
		md := metadata.New(map[string]string{"x-real-ip": s.grpcIP.String()})
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	return retry.Do(ctx, retry.IsGRPCRetriable, func() error {
		callCtx, cancel := context.WithTimeout(ctx, timeoutCall)
		defer cancel()

		_, err := s.grpcClient.UpdateMetrics(callCtx, pb.UpdateMetricsRequest_builder{
			Metrics: metricsProto,
		}.Build())

		if err != nil {
			return fmt.Errorf("не удалось отправить метрики: %w", err)
		}

		return nil
	})
}

// URLBuilder собирает относительный путь для отправки метрики через
// [Sender.SendMetrics].
//
// Возвращает строку вида /update/{metricType}/{name}/{value}.
func URLBuilder(metricType, name, value string) string {
	return fmt.Sprintf("/update/%s/%s/%s", metricType, name, value)
}

// getLocalIP возвращает локальный IP-адрес, с которого ОС отправляет
// пакеты на serverAddr (host:port).
//
// UDP-«соединение» только выбирает маршрут и привязывает локальный адрес:
// пакеты не отправляются, поэтому сервер может быть недоступен. При
// нескольких сетевых интерфейсах возвращается адрес того, через который
// идёт маршрут к серверу, то есть именно тот, что увидит сервер.
func getLocalIP(serverAddr string) (net.IP, error) {
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}

// createGRPCClient создаёт TLS-соединение с gRPC-сервером addr и клиент
// сервиса Metrics поверх него.
//
// Сертификат из certFile загружается в пул доверенных корней: агент
// принимает только сервер, предъявивший этот сертификат, а имя для проверки
// берётся из addr. Возвращает ошибку, если файл не читается или в нём нет
// PEM-сертификата.
//
// Соединение устанавливается лениво, при первом вызове, поэтому сервер
// может быть ещё не запущен. Переподключение идёт с экспоненциальной
// паузой [backoff.DefaultConfig], на одну попытку подключения отводится не
// меньше timeoutCall.
func createGRPCClient(addr, certFile string) (pb.MetricsClient, *grpc.ClientConn, error) {
	var p grpc.ConnectParams
	p.MinConnectTimeout = timeoutCall
	p.Backoff = backoff.DefaultConfig

	caCert, err := os.ReadFile(certFile)
	if err != nil {
		return nil, nil, fmt.Errorf("не удалось прочитать сертификат %s: %w", certFile, err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, nil, fmt.Errorf("в файле %s нет корректного PEM-сертификата", certFile)
	}

	creds := credentials.NewClientTLSFromCert(pool, "")

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds), grpc.WithConnectParams(p))
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка при установлении соединения с сервером: %w", err)
	}

	c := pb.NewMetricsClient(conn)

	return c, conn, nil
}
