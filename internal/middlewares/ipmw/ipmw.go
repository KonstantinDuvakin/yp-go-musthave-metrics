// Package ipmw предоставляет HTTP-middleware, пропускающее только запросы
// агентов из доверенной подсети.
//
// IP-адрес агента берётся из заголовка X-Real-IP, который агент
// выставляет сам.
package ipmw

import (
	"net"
	"net/http"
)

// Middleware возвращает middleware, проверяющее, что IP-адрес из заголовка
// X-Real-IP входит в подсеть ipNet.
//
// Если заголовка нет, в нём не IP-адрес или адрес не входит в ipNet,
// запрос отклоняется со статусом 403 Forbidden и до хендлера не доходит.
// ipNet не должен быть nil: при пустой доверенной подсети middleware
// подключать не нужно.
func Middleware(ipNet *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			xRealIPHeader := r.Header.Get("X-Real-IP")

			if xRealIPHeader == "" {
				http.Error(rw, "IP isn't provided", http.StatusForbidden)
				return
			}

			ip := net.ParseIP(xRealIPHeader)

			if !ipNet.Contains(ip) {
				http.Error(rw, "provided IP is not allowed", http.StatusForbidden)
				return
			}

			next.ServeHTTP(rw, r)
		})
	}
}
