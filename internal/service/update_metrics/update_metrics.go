// Package updatemetrics содержит сервисный слой приёма пакета метрик,
// общий для HTTP и gRPC: сохранение пакета в хранилище и запись события
// аудита.
//
// Транспорты только преобразуют входные данные в []models.Metrics и
// вызывают [Service.UpdateBatch], поэтому правила сохранения и аудита живут
// в одном месте.
package updatemetrics

import (
	"context"
	"fmt"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

// AuditSender принимает события аудита. Ему соответствует, например,
// sendtoaudit.AuditService; в тестах подставляется заглушка.
type AuditSender interface {
	SendEvent(models.Audit)
}

// Service принимает пакеты метрик: сохраняет их и пишет событие аудита.
// Создаётся через [New] и используется обоими транспортами.
type Service struct {
	store       storage.MetricsStorage
	auditSender AuditSender
}

// New создаёт [Service], сохраняющий метрики в store и отправляющий события
// аудита в auditSender. Оба аргумента не должны быть nil.
func New(store storage.MetricsStorage, auditSender AuditSender) *Service {
	return &Service{store: store, auditSender: auditSender}
}

// UpdateBatch сохраняет пакет metrics одним вызовом SaveMetricsBatch и,
// если сохранение прошло успешно, отправляет событие аудита: время в
// миллисекундах, имена метрик пакета и ip отправителя.
//
// Если сохранить не удалось, аудит не пишется, а возвращается обёрнутая
// ошибка хранилища (доступна через errors.Is). Сам метод в лог не пишет:
// это делает транспорт.
func (s *Service) UpdateBatch(ctx context.Context, metrics []models.Metrics, ip string) error {
	err := s.store.SaveMetricsBatch(ctx, metrics)
	if err != nil {
		return fmt.Errorf("не удалось сохранить метрики: %w", err)
	}

	event := models.Audit{
		TS:        time.Now().UnixMilli(),
		Metrics:   make([]string, 0, len(metrics)),
		IPAddress: ip,
	}

	for _, el := range metrics {
		event.Metrics = append(event.Metrics, el.ID)
	}

	s.auditSender.SendEvent(event)

	return nil
}
