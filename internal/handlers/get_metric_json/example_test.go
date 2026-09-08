package getmetricjson

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

// Получение значения метрики через эндпоинт POST /value: в запросе передаются
// id и type, в ответе — метрика с заполненным значением.
func ExampleGetMetricJSON() {
	store := memstorage.NewMemStorage()
	_ = store.SetGauge("Alloc", 42.5)

	srv := httptest.NewServer(GetMetricJSON(store))
	defer srv.Close()

	body := `{"id":"Alloc","type":"gauge"}`
	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	fmt.Println("статус:", resp.StatusCode)
	fmt.Print(string(data))

	// Output:
	// статус: 200
	// {"id":"Alloc","type":"gauge","value":42.5}
}
