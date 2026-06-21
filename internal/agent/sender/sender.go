package sender

import (
	"context"
	"fmt"
	"time"

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
	_, err := s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post("/update")
	return err
}

func URLBuilder(metricType, name, value string) string {
	return fmt.Sprintf("/update/%s/%s/%s", metricType, name, value)
}
