package sender

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type Sender struct {
	client *resty.Client
}

func NewSender(baseUrl string) *Sender {
	client := resty.New()
	client.SetBaseURL("http://" + baseUrl)
	client.SetHeader("Content-Type", "text/plain")
	client.SetTimeout(2 * time.Second)
	return &Sender{
		client: client,
	}
}

func (s Sender) SendMetrics(ctx context.Context, url string) error {
	_, err := s.client.R().SetContext(ctx).Post(url)
	return err
}

func URLBuilder(metricType, name, value string) string {
	return fmt.Sprintf("/update/%s/%s/%s", metricType, name, value)
}
