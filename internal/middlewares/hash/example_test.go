package hash

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	hashhelper "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/hash"
)

// HashMiddleware подписывает ответ ключом и кладёт подпись в заголовок
// HashSHA256. Клиент может проверить целостность тела тем же ключом.
func ExampleHashMiddleware() {
	key := "secret-key"

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	srv := httptest.NewServer(HashMiddleware(key)(final))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	want := hashhelper.CreateHeaderHash(body, key)

	fmt.Println("подпись совпадает:", resp.Header.Get("HashSHA256") == want)

	// Output:
	// подпись совпадает: true
}
