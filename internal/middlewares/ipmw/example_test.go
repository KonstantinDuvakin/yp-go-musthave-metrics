package ipmw

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
)

// Middleware пропускает запрос, только если IP из заголовка X-Real-IP
// входит в доверенную подсеть, иначе отвечает 403.
func ExampleMiddleware() {
	_, trusted, _ := net.ParseCIDR("192.168.1.0/24")

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := Middleware(trusted)(final)

	for _, ip := range []string{"192.168.1.10", "10.0.0.1"} {
		req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
		req.Header.Set("X-Real-IP", ip)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
		fmt.Println(ip, rec.Code)
	}

	// Output:
	// 192.168.1.10 200
	// 10.0.0.1 403
}
