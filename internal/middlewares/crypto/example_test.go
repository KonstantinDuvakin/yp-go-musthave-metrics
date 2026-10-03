package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	cryptohelper "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/crypto"
)

// Middleware расшифровывает тело запроса с заголовком X-Encrypted: true,
// и хендлер получает уже исходные данные.
func ExampleMiddleware() {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		fmt.Println("хендлер получил:", string(body))
	})

	srv := httptest.NewServer(Middleware(privateKey)(final))
	defer srv.Close()

	encrypted, err := cryptohelper.EncryptBodyWithKey([]byte(`{"id":"Alloc"}`), &privateKey.PublicKey)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader(encrypted))
	req.Header.Set("X-Encrypted", "true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("статус:", resp.StatusCode)

	// Output:
	// хендлер получил: {"id":"Alloc"}
	// статус: 200
}
