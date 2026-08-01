package pingDBHandler

import (
	"context"
	"net/http"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

func PingDBHandler(pinger Pinger) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if pinger == nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := pinger.Ping(r.Context()); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("OK"))
	}
}
