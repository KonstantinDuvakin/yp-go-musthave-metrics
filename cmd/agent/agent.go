package main

import (
	"context"
	"sync"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/collector"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/agent_storage"
	"go.uber.org/zap"
)

type Agent struct {
	storage *agent_storage.AgentStorage
	sender  *sender.Sender
}

func New(storage *agent_storage.AgentStorage, sender *sender.Sender) *Agent {
	return &Agent{
		storage: storage,
		sender:  sender,
	}
}

func (a *Agent) Run(ctx context.Context, poll, report time.Duration, hashKey string, limit int) {
	pollTicker := time.NewTicker(poll)
	defer pollTicker.Stop()

	gopsPollTicker := time.NewTicker(poll)
	defer gopsPollTicker.Stop()

	reportTicker := time.NewTicker(report)
	defer reportTicker.Stop()

	var wg sync.WaitGroup

	jobs := make(chan []models.Metrics, limit)

	for i := 0; i < limit; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for job := range jobs {
				err := a.sender.SendMetricsBatch(ctx, job, hashKey)
				if err != nil {
					logger.Log.Error("Couldn't sent metric\nError: \n", zap.Error(err))
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				for name, value := range collector.Collector() {
					a.storage.SetGauge(name, value)
				}
				a.storage.AddCounter("PollCount")
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case <-gopsPollTicker.C:
				for name, value := range collector.GopsCollector() {
					a.storage.SetGauge(name, value)
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(jobs)

		for {
			select {
			case <-ctx.Done():
				return
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

				select {
				case jobs <- metricsBatch:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	wg.Wait()
}
