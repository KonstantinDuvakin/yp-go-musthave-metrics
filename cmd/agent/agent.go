package main

import (
	"context"
	"crypto/rsa"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/collector"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/agent_storage"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

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
// Контексты разделены по назначению:
//   - ctx управляет сбором: по его отмене опрос прекращается, в очередь
//     ставится финальный снимок метрик, и очередь закрывается;
//   - sendCtx используется для HTTP-отправки: пока он не отменён, воркеры
//     досылают все пакеты из очереди, в том числе после отмены ctx. Отмена
//     sendCtx прерывает текущие запросы и ожидание места в очереди.
//
// Ограничивать время досылки — задача вызывающего кода: отменить sendCtx,
// если Run не вернул управление за отведённое время.
//
// Run возвращает управление, когда завершились все горутины, и возвращает
// первую ошибку, полученную от них (сейчас горутины ошибок не возвращают,
// ошибки отправки только логируются).
func (a *Agent) Run(ctx, sendCtx context.Context, poll, report time.Duration, hashKey string, limit int, pubKey *rsa.PublicKey) error {
	pollTicker := time.NewTicker(poll)
	defer pollTicker.Stop()

	gopsPollTicker := time.NewTicker(poll)
	defer gopsPollTicker.Stop()

	reportTicker := time.NewTicker(report)
	defer reportTicker.Stop()

	eg, gctx := errgroup.WithContext(ctx)
	jobs := make(chan []models.Metrics, limit)

	for i := 0; i < limit; i++ {
		eg.Go(func() error {
			for job := range jobs {
				err := a.sender.SendMetricsBatch(sendCtx, job, hashKey, pubKey)
				if err != nil {
					logger.Log.Error("Couldn't sent metric\nError: \n", zap.Error(err))
				}
			}
			return nil
		})
	}

	eg.Go(func() error {
		for {
			select {
			case <-gctx.Done():
				return nil
			case <-pollTicker.C:
				for name, value := range collector.Collector() {
					a.storage.SetGauge(name, value)
				}
				a.storage.AddCounter("PollCount")
			}
		}
	})

	eg.Go(func() error {
		for {
			select {
			case <-gctx.Done():
				return nil
			case <-gopsPollTicker.C:
				for name, value := range collector.GopsCollector() {
					a.storage.SetGauge(name, value)
				}
			}
		}
	})

	eg.Go(func() error {
		defer close(jobs)

		sendFinal := func() {
			mb := a.storage.CollectMetrics()
			if mb == nil {
				return
			}
			select {
			case jobs <- mb:
			case <-sendCtx.Done():
			}
		}

		for {
			select {
			case <-gctx.Done():
				sendFinal()
				return nil
			case <-reportTicker.C:
				metricsBatch := a.storage.CollectMetrics()

				if metricsBatch == nil {
					continue
				}

				select {
				case jobs <- metricsBatch:
				case <-gctx.Done():
					sendFinal()
					return nil
				}
			}
		}
	})

	return eg.Wait()
}
