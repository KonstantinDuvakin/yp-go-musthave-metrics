// Package sender предназначен для отправки метрик на сервер по HTTP.
//
// Пакет инкапсулирует resty-клиент и предоставляет методы для отправки
// метрик как по URL-параметрам, так и в формате JSON (с gzip-сжатием,
// подписью тела запроса и повторными попытками при временных ошибках).
package sender

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/hash"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/retry"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/go-resty/resty/v2"
)

// Sender отправляет метрики на сервер сбора метрик.
//
// Хранит настроенный HTTP-клиент [resty.Client] с заданными базовым URL,
// заголовками и таймаутом. Создаётся через [NewSender].
type Sender struct {
	client *resty.Client
}

// NewSender создаёт [Sender] с готовым к работе resty-клиентом.
//
// baseURL — адрес сервера без схемы (например, "localhost:8080"); схема
// http:// добавляется автоматически. Клиенту задаётся таймаут 2 секунды
// и заголовок Content-Type: text/plain по умолчанию.
func NewSender(baseURL string) *Sender {
	client := resty.New()
	client.SetBaseURL("http://" + baseURL)
	client.SetHeader("Content-Type", "text/plain")
	client.SetTimeout(2 * time.Second)
	return &Sender{
		client: client,
	}
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

// SendMetricsBatch отправляет пакет метрик в формате JSON на эндпоинт /updates.
//
// Работает аналогично [Sender.SendMetricsJSON], но принимает срез метрик и
// отправляет их одним запросом: тело сжимается gzip, при непустом hashKey
// добавляется подпись HashSHA256, а сам запрос повторяется при временных
// ошибках. Возвращает ошибку, если сервер ответил статусом >= 400.
func (s *Sender) SendMetricsBatch(ctx context.Context, metrics []models.Metrics, hashKey string) error {
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

	return retry.Do(ctx, retry.IsHTTPRetriable, func() error {
		req := s.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(bufGZip.Bytes())

		if hashKey != "" {
			req.SetHeader("HashSHA256", sign)
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

// URLBuilder собирает относительный путь для отправки метрики через
// [Sender.SendMetrics].
//
// Возвращает строку вида /update/{metricType}/{name}/{value}.
func URLBuilder(metricType, name, value string) string {
	return fmt.Sprintf("/update/%s/%s/%s", metricType, name, value)
}
