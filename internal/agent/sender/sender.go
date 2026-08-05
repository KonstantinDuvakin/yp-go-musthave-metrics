package sender

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/retry"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/go-resty/resty/v2"
)

type Sender struct {
	client *resty.Client
}

func NewSender(baseURL string) *Sender {
	client := resty.New()
	client.SetBaseURL("http://" + baseURL)
	client.SetHeader("Content-Type", "text/plain")
	client.SetTimeout(2 * time.Second)
	return &Sender{
		client: client,
	}
}

func (s *Sender) SendMetrics(ctx context.Context, url string) error {
	_, err := s.client.R().SetContext(ctx).Post(url)
	return err
}

func (s *Sender) SendMetricsJson(ctx context.Context, body models.Metrics) error {
	bufGZip := bytes.NewBuffer(nil)
	zw := gzip.NewWriter(bufGZip)
	if err := json.NewEncoder(zw).Encode(body); err != nil {
		return err
	}

	if err := zw.Close(); err != nil {
		return err
	}

	_, err := s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(bufGZip.Bytes()).
		Post("/update")
	return err
}

func (s *Sender) SendMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	bufGZip := bytes.NewBuffer(nil)
	zw := gzip.NewWriter(bufGZip)
	if err := json.NewEncoder(zw).Encode(metrics); err != nil {
		return err
	}

	if err := zw.Close(); err != nil {
		return err
	}

	return retry.Do(ctx, retry.IsHttpRetriable, func() error {
		resp, err := s.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(bufGZip.Bytes()).
			Post("/updates")

		if err != nil {
			return err
		}

		if resp.IsError() {
			return fmt.Errorf("batch upload failed: status %d", resp.StatusCode())
		}

		return nil
	})
}

func URLBuilder(metricType, name, value string) string {
	return fmt.Sprintf("/update/%s/%s/%s", metricType, name, value)
}
