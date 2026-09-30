package main

import (
	"context"
	"crypto/rsa"
	"sync"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/collector"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/agent_storage"
	"go.uber.org/zap"
)

// closeTimeout ограничивает время, которое агент при остановке ждёт досылки
// оставшихся метрик на сервер. По его истечении незавершённые запросы
// прерываются, а неотправленные данные теряются.
const closeTimeout = 5 * time.Second

// Agent связывает хранилище собранных метрик и отправитель на сервер.
type Agent struct {
	storage *agentstorage.AgentStorage
	sender  *sender.Sender
}

// New создаёт [Agent] с заданными хранилищем и отправителем.
func New(storage *agentstorage.AgentStorage, sender *sender.Sender) *Agent {
	return &Agent{
		storage: storage,
		sender:  sender,
	}
}

// Run запускает рабочие циклы агента и блокируется до отмены ctx и
// завершения отправки оставшихся данных.
//
// Метрики рантайма и системы опрашиваются с интервалом poll, накопленный
// пакет отправляется на сервер с интервалом report. Отправку выполняют
// limit параллельных воркеров; hashKey, если не пуст, используется для
// подписи запросов, pubKey, если не nil, — для шифрования тела.
//
// По отмене ctx опрос прекращается, в очередь ставится финальный снимок
// метрик, а воркеры досылают все поставленные в очередь пакеты. Отправка
// при остановке не зависит от ctx и ограничена closeTimeout: по его истечении
// незавершённые запросы прерываются и Run возвращает управление.
func (a *Agent) Run(ctx context.Context, poll, report time.Duration, hashKey string, limit int, pubKey *rsa.PublicKey) {
	pollTicker := time.NewTicker(poll)
	defer pollTicker.Stop()

	gopsPollTicker := time.NewTicker(poll)
	defer gopsPollTicker.Stop()

	reportTicker := time.NewTicker(report)
	defer reportTicker.Stop()

	innerCtx, cancel := context.WithCancel(context.Background())

	cls := make(chan struct{})

	var wg sync.WaitGroup

	jobs := make(chan []models.Metrics, limit)

	for i := 0; i < limit; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for job := range jobs {
				err := a.sender.SendMetricsBatch(innerCtx, job, hashKey, pubKey)
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
			case <-cls:
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
			case <-cls:
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

		sendFinal := func() {
			mb := a.storage.CollectMetrics()
			if mb == nil {
				return
			}
			select {
			case jobs <- mb:
			case <-innerCtx.Done():
			}
		}

		for {
			select {
			case <-cls:
				sendFinal()
				return
			case <-reportTicker.C:
				metricsBatch := a.storage.CollectMetrics()

				if metricsBatch == nil {
					continue
				}

				select {
				case jobs <- metricsBatch:
				case <-cls:
					sendFinal()
					return
				}
			}
		}
	}()

	wgChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(wgChan)
	}()

	<-ctx.Done()
	close(cls)

	select {
	case <-time.After(closeTimeout):
		cancel()
	case <-wgChan:
		cancel()
	}
}
