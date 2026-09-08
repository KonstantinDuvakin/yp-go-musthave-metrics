package updatehandler

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
	"github.com/go-chi/chi/v5"
)

// Обновление метрики через эндпоинт POST /update/{type}/{name}/{value}.
// Параметры метрики передаются в пути URL, поэтому обработчик монтируется в
// chi-роутер, извлекающий эти параметры.
func ExampleUpdateHandler() {
	store := memstorage.NewMemStorage()

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", UpdateHandler(store))

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/update/counter/PollCount/10", "text/plain", nil)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("статус:", resp.StatusCode)

	v, ok, _ := store.GetCounter("PollCount")
	fmt.Printf("PollCount=%d найдена=%v\n", v, ok)

	// Output:
	// статус: 200
	// PollCount=10 найдена=true
}
