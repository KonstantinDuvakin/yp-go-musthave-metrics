package sender

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
)

// Отправка одной метрики в формате JSON на эндпоинт /update.
func ExampleSender_SendMetricsJSON() {
	// Поднимаем тестовый сервер, имитирующий сервер сбора метрик.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("получен запрос:", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// srv.URL имеет вид http://127.0.0.1:PORT — NewSender ждёт адрес без схемы.
	s := NewSender(strings.TrimPrefix(srv.URL, "http://"))

	value := 42.5
	metric := models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &value}

	if err := s.SendMetricsJSON(context.Background(), metric, ""); err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	fmt.Println("метрика отправлена")

	// Output:
	// получен запрос: POST /update
	// метрика отправлена
}

// Отправка пакета метрик на эндпоинт /updates.
func ExampleSender_SendMetricsBatch() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("получен запрос:", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewSender(strings.TrimPrefix(srv.URL, "http://"))

	value := 42.5
	delta := int64(10)
	batch := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &value},
		{ID: "PollCount", MType: models.Counter, Delta: &delta},
	}

	if err := s.SendMetricsBatch(context.Background(), batch, ""); err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	fmt.Println("пакет отправлен")

	// Output:
	// получен запрос: POST /updates
	// пакет отправлен
}

// URLBuilder собирает относительный путь для отправки метрики через URL.
func ExampleURLBuilder() {
	url := URLBuilder(models.Gauge, "Alloc", "42.5")
	fmt.Println(url)

	// Output:
	// /update/gauge/Alloc/42.5
}
