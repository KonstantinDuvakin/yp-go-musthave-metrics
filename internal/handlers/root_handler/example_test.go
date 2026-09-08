package roothandler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage/mem_storage"
)

// Корневая страница GET / отдаёт список всех метрик в формате HTML.
func ExampleRootHandler() {
	store := memstorage.NewMemStorage()
	_ = store.SetGauge("Alloc", 42.5)

	srv := httptest.NewServer(RootHandler(store))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("статус:", resp.StatusCode)
	fmt.Println("есть метрика Alloc:", strings.Contains(string(body), "Alloc: 42.500000"))

	// Output:
	// статус: 200
	// есть метрика Alloc: true
}
