package main

import (
	"context"
	"fmt"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/collector"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/agentStorage"
)

type Agent struct {
	storage *agentStorage.AgentStorage
	sender  *sender.Sender
}

func New(storage *agentStorage.AgentStorage, sender *sender.Sender) *Agent {
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
			gauges, counters := a.storage.Snapshot()
			metricsBatch := make([]models.Metrics, 0, len(gauges)+len(counters))

			for name, value := range gauges {
				gaugeMetric := models.Metrics{
					ID:    name,
					MType: models.Gauge,
					Value: &value,
				}

				metricsBatch = append(metricsBatch, gaugeMetric)
			}

			for name, value := range counters {
				counterMetric := models.Metrics{
					ID:    name,
					MType: models.Counter,
					Delta: &value,
				}

				metricsBatch = append(metricsBatch, counterMetric)
			}

			if len(metricsBatch) == 0 {
				continue
			}

			err := a.sender.SendMetricsBatch(ctx, metricsBatch)
			if err != nil {
				fmt.Printf("Couldn't sent metrics\nError: %v\n", err)
			}
			fmt.Println("Send")
		}
	}
}
