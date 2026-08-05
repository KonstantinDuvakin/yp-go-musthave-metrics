package updateBatchMetrics

import (
	"encoding/json"
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/model"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"go.uber.org/zap"
)

func UpdateBatchMetrics(store storage.MetricsStorage) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var req []models.Metrics

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&req); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			rw.Write([]byte("Invalid json"))
			return
		}

		if err := store.SaveMetricsBatch(r.Context(), req); err != nil {
			logger.Log.Error("Failed to save metrics batch: ", zap.Error(err))
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte("Couldn't save metrics batch"))
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
