package pingdbhandler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

// okPinger — заглушка источника данных, всегда доступного.
type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

// Проверка доступности БД через эндпоинт GET /ping.
func ExamplePingDBHandler() {
	srv := httptest.NewServer(PingDBHandler(okPinger{}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("статус:", resp.StatusCode)
	fmt.Println("тело:", string(body))

	// Output:
	// статус: 200
	// тело: OK
}
