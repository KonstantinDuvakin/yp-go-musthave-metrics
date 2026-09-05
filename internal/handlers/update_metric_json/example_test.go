package updatemetricjson

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

// Обновление одной метрики через эндпоинт POST /update с телом в формате JSON.
func ExampleUpdateMetricJSON() {
	store := memstorage.NewMemStorage()

	srv := httptest.NewServer(UpdateMetricJSON(store))
	defer srv.Close()

	body := `{"id":"Alloc","type":"gauge","value":42.5}`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("статус:", resp.StatusCode)

	// Проверяем, что значение сохранилось в хранилище.
	v, ok, _ := store.GetGauge("Alloc")
	fmt.Printf("Alloc=%v найдена=%v\n", v, ok)

	// Output:
	// статус: 200
	// Alloc=42.5 найдена=true
}
