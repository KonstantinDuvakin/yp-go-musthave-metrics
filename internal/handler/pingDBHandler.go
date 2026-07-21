package handler

import (
	"net/http"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Pinger interface {
	Ping() error
}

func PingDBHandler(pinger Pinger) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		if err := pinger.Ping(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("OK"))
	}
}
