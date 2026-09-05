// Package sendtoaudit асинхронно доставляет события аудита в файл и/или на
// внешний URL через фоновые горутины.
package sendtoaudit

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

// AuditService асинхронно записывает события аудита в файл и/или отправляет
// их на HTTP-эндпоинт. Каждое направление обслуживается своей горутиной,
// запускаемой методом [AuditService.Start].
type AuditService struct {
	filename string
	url      string
	fileCh   chan models.Audit
	urlCh    chan models.Audit
	doneCh   chan struct{}
	client   *http.Client
	wg       *sync.WaitGroup
}

type auditFunc func(auditData models.Audit) error

// NewAuditService создаёт [AuditService]. Направление доставки включается
// непустым аргументом: auditFile — путь к файлу аудита, auditURL — адрес
// HTTP-эндпоинта. Пустой аргумент отключает соответствующее направление.
func NewAuditService(auditFile, auditURL string) *AuditService {
	var fileCh chan models.Audit
	var urlCh chan models.Audit

	if auditFile != "" {
		fileCh = make(chan models.Audit, 1)
	}

	if auditURL != "" {
		urlCh = make(chan models.Audit, 1)
	}

	return &AuditService{
		filename: auditFile,
		url:      auditURL,
		fileCh:   fileCh,
		urlCh:    urlCh,
		doneCh:   make(chan struct{}),
		client:   &http.Client{Timeout: time.Second * 5},
		wg:       &sync.WaitGroup{},
	}
}

// Start запускает фоновые горутины доставки событий. Вызывается один раз
// перед использованием [AuditService.SendEvent].
func (a *AuditService) Start() {
	a.wg.Add(2)
	go a.auditWriter(a.fileCh, a.auditToFile)
	go a.auditWriter(a.urlCh, a.auditToURL)
}

// Stop сигнализирует горутинам о завершении, дожидается обработки уже
// накопленных событий и возвращает управление. После вызова сервис
// использовать нельзя.
func (a *AuditService) Stop() {
	close(a.doneCh)
	a.wg.Wait()
}

// SendEvent передаёт событие аудита во включённые направления доставки.
// Метод неблокирующий по завершении сервиса: после [AuditService.Stop]
// событие может быть отброшено. Подходит на роль audit-функции для
// [updatebatchmetrics.UpdateBatchMetrics].
func (a *AuditService) SendEvent(event models.Audit) {
	if a.fileCh != nil {
		select {
		case a.fileCh <- event:
		case <-a.doneCh:
		}
	}

	if a.urlCh != nil {
		select {
		case a.urlCh <- event:
		case <-a.doneCh:
		}
	}
}

func (a *AuditService) auditWriter(ch chan models.Audit, write auditFunc) {
	defer a.wg.Done()

	if ch == nil {
		return
	}

	for {
		select {
		case event := <-ch:
			err := write(event)
			if err != nil {
				logger.Log.Error("Ошибка аудита: ", zap.Error(err))
			}
		case <-a.doneCh:
			for {
				select {
				case e := <-ch:
					err := write(e)
					if err != nil {
						logger.Log.Error("Ошибка аудита: ", zap.Error(err))
					}
				default:
					return
				}
			}
		}
	}
}

func (a *AuditService) auditToFile(auditData models.Audit) error {
	file, err := os.OpenFile(a.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer file.Close()

	enc := json.NewEncoder(file)

	if err = enc.Encode(auditData); err != nil {
		return err
	}

	return nil
}

func (a *AuditService) auditToURL(auditData models.Audit) error {
	buf := bytes.NewBuffer(nil)

	enc := json.NewEncoder(buf)

	if err := enc.Encode(auditData); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, a.url, buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		if resp.StatusCode >= http.StatusInternalServerError {
			return errors.New("сервис ответил ошибкой")
		}
		return errors.New("ошибка отправки данных")
	}

	return nil
}
