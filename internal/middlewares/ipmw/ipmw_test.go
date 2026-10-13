package ipmw

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustParseCIDR(t *testing.T, s string) *net.IPNet {
	t.Helper()
	_, ipNet, err := net.ParseCIDR(s)
	require.NoError(t, err)
	return ipNet
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		subnet     string
		realIP     string // пусто — заголовок не выставляется
		wantCode   int
		wantCalled bool
	}{
		{name: "ip in subnet", subnet: "192.168.1.0/24", realIP: "192.168.1.10", wantCode: http.StatusOK, wantCalled: true},
		{name: "network address in subnet", subnet: "192.168.1.0/24", realIP: "192.168.1.0", wantCode: http.StatusOK, wantCalled: true},
		{name: "broadcast address in subnet", subnet: "192.168.1.0/24", realIP: "192.168.1.255", wantCode: http.StatusOK, wantCalled: true},
		{name: "ip outside subnet", subnet: "192.168.1.0/24", realIP: "192.168.2.10", wantCode: http.StatusForbidden},
		{name: "loopback in /8", subnet: "127.0.0.0/8", realIP: "127.0.0.1", wantCode: http.StatusOK, wantCalled: true},
		{name: "single host /32 match", subnet: "10.0.0.5/32", realIP: "10.0.0.5", wantCode: http.StatusOK, wantCalled: true},
		{name: "single host /32 mismatch", subnet: "10.0.0.5/32", realIP: "10.0.0.6", wantCode: http.StatusForbidden},
		{name: "ipv6 in subnet", subnet: "fd00::/8", realIP: "fd00::1", wantCode: http.StatusOK, wantCalled: true},
		{name: "ipv6 against ipv4 subnet", subnet: "127.0.0.0/8", realIP: "::1", wantCode: http.StatusForbidden},
		{name: "missing header", subnet: "192.168.1.0/24", realIP: "", wantCode: http.StatusForbidden},
		{name: "not an ip", subnet: "192.168.1.0/24", realIP: "not-an-ip", wantCode: http.StatusForbidden},
		{name: "ip with port", subnet: "192.168.1.0/24", realIP: "192.168.1.10:8080", wantCode: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})
			h := Middleware(mustParseCIDR(t, tt.subnet))(next)

			req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
			if tt.realIP != "" {
				req.Header.Set("X-Real-IP", tt.realIP)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			require.Equal(t, tt.wantCode, rec.Code)
			require.Equal(t, tt.wantCalled, called, "хендлер вызывается только для разрешённых IP")
		})
	}
}
