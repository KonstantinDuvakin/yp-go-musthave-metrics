package handler

import (
	"fmt"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

func RootHandler(storage *storage.MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gauges := storage.GetAllGauges()
		counters := storage.GetAllCounters()

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		fmt.Fprintln(w, "<html><body><h1>Metrics</h1><ul>")

		for name, value := range gauges {
			fmt.Fprintf(w, "<li>%s: %f</li>", name, value)
		}

		for name, value := range counters {
			fmt.Fprintf(w, "<li>%s: %d</li>", name, value)
		}

		fmt.Fprintln(w, "</ul></body></html>")
	}
}
