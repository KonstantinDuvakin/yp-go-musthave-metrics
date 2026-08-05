package rootHandler

import (
	"fmt"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

func RootHandler(storage storage.MetricsStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gauges, err := storage.GetAllGauges()
		if err != nil {
			logger.Log.Error("error getting gauges: ", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		counters, err := storage.GetAllCounters()
		if err != nil {
			logger.Log.Error("error getting counters: ", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

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
