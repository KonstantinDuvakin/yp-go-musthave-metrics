package main

import (
	"context"
	"fmt"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/collector"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

type Agent struct {
	storage *storage.AgentStorage
	sender  *sender.Sender
}

func New(storage *storage.AgentStorage, sender *sender.Sender) *Agent {
	return &Agent{
		storage: storage,
		sender:  sender,
	}
}

func (a *Agent) Run(ctx context.Context, poll, report time.Duration) {
	pollTicker := time.NewTicker(poll)
	defer pollTicker.Stop()

	reportTicker := time.NewTicker(report)
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			for name, value := range collector.Collector() {
				a.storage.SetGauge(name, value)
			}
			a.storage.AddCounter("PollCount")
		case <-reportTicker.C:
			gauge, counter := a.storage.Snapshot()
			for name, value := range gauge {
				body := models.Metrics{
					ID:    name,
					MType: models.Gauge,
					Value: &value,
				}

				err := a.sender.SendMetricsJson(ctx, body)
				if err != nil {
					fmt.Printf("Couldn't sent gauge metric %s\nError: %v\n", name, err)
				}
			}
			for name, value := range counter {
				body := models.Metrics{
					ID:    name,
					MType: models.Counter,
					Delta: &value,
				}
				err := a.sender.SendMetricsJson(ctx, body)
				if err != nil {
					fmt.Printf("Couldn't sent counter metric %s\nError: %v\n", name, err)
				}
			}
			fmt.Println("Send")
		}
	}
}
