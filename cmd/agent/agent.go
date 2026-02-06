package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/collector"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
	models "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

type Agent struct {
	storage *storage.AgentStorage
	address string
}

func New(storage *storage.AgentStorage, address string) *Agent {
	return &Agent{
		storage: storage,
		address: address,
	}
}

func (a *Agent) Run(poll, report time.Duration) {
	pollTicker := time.NewTicker(poll)
	reportTicker := time.NewTicker(report)

	for {
		select {
		case <-pollTicker.C:
			for name, value := range collector.Collector() {
				a.storage.SetGauge(name, value)
			}
			a.storage.AddCounter("PollCount")
		case <-reportTicker.C:
			gauge, counter := a.storage.Snapshot()
			for name, value := range gauge {
				url := sender.URLBuilder(a.address, models.Gauge, name, strconv.FormatFloat(value, 'g', -1, 64))
				go sender.SendMetrics(url)
			}
			for name, value := range counter {
				url := sender.URLBuilder(a.address, models.Counter, name, strconv.FormatInt(value, 10))
				go sender.SendMetrics(url)
			}
			fmt.Println("Send")
		}
	}
}
