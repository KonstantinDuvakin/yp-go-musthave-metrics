package getmetrichandler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
	"github.com/go-chi/chi/v5"
)

// Получение значения метрики через эндпоинт GET /value/{type}/{name}.
// Значение возвращается обычным текстом.
func ExampleGetMetricHandler() {
	store := memstorage.NewMemStorage()
	_ = store.SetGauge("Alloc", 42.5)

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", GetMetricHandler(store))

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/value/gauge/Alloc")
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	fmt.Println("статус:", resp.StatusCode)
	fmt.Println("значение:", string(data))

	// Output:
	// статус: 200
	// значение: 42.5
}
