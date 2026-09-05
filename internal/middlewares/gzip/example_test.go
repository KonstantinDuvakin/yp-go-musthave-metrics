package gzip

import (
	gz "compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

// Middleware сжимает ответ, если клиент прислал Accept-Encoding: gzip.
func ExampleMiddleware() {
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("metrics response"))
	})

	srv := httptest.NewServer(Middleware(final))
	defer srv.Close()

	// Явно просим gzip. При заданном вручную Accept-Encoding транспорт Go не
	// распаковывает ответ автоматически — сделаем это сами.
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Content-Encoding:", resp.Header.Get("Content-Encoding"))

	zr, _ := gz.NewReader(resp.Body)
	body, _ := io.ReadAll(zr)
	fmt.Println("тело:", string(body))

	// Output:
	// Content-Encoding: gzip
	// тело: metrics response
}
