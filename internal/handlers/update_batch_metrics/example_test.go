package updatebatchmetrics

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	updatemetrics "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/update_metrics"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

// Пакетное обновление метрик через эндпоинт POST /updates. Обработчик
// получает сервис [updatemetrics.Service], который сохраняет пакет и пишет
// событие аудита.
func ExampleUpdateBatchMetrics() {
	store := memstorage.NewMemStorage()

	audit := auditFunc(func(event models.Audit) {
		fmt.Println("аудит, метрик принято:", len(event.Metrics))
	})

	srv := httptest.NewServer(UpdateBatchMetrics(updatemetrics.New(store, audit)))
	defer srv.Close()

	body := `[
		{"id":"Alloc","type":"gauge","value":42.5},
		{"id":"PollCount","type":"counter","delta":10}
	]`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("статус:", resp.StatusCode)

	// Output:
	// аудит, метрик принято: 2
	// статус: 200
}
