package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/getMetricHandler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/getMetricJson"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/pingDBHandler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/rootHandler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/updateBatchMetrics"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/updateHandler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/updateMetricJson"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/gzip"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/hash"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/sendToAudit"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/serverStorage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	c := config.NewConfigServer()
	flag.Parse()
	c.ApplyEnv()

	err := logger.InitializeLogger("info")
	if err != nil {
		logger.Log.Warn("Failed to initialize logger", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, db, shutdown, err := serverStorage.NewStorage(ctx, c)
	if err != nil {
		logger.Log.Fatal("Failed to initialize storage", zap.Error(err))
	}

	var pinger pingDBHandler.Pinger
	if db != nil {
		pinger = db
	}

	auditDoneCh := make(chan struct{})
	auditService := sendToAudit.NewAuditService(c.AuditFile, c.AuditUrl)
	go func() {
		auditService.Start()
		close(auditDoneCh)
	}()

	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Use(logger.RequestLogger)
		r.Use(gzip.Middleware)
		r.Use(hash.HashMiddleware(c.Key))

		r.Get("/", rootHandler.RootHandler(store))
		r.Route("/update", func(r chi.Router) {
			r.Post("/", updateMetricJson.UpdateMetricJson(store))
			r.Post(`/{type}/{name}/{value}`, updateHandler.UpdateHandler(store))
		})
		r.Route("/updates", func(r chi.Router) {
			r.Post("/", updateBatchMetrics.UpdateBatchMetrics(store, auditService.SendEvent))
		})
		r.Route("/value", func(r chi.Router) {
			r.Post("/", getMetricJson.GetMetricJson(store))
			r.Get(`/{type}/{name}`, getMetricHandler.GetMetricHandler(store))
		})
		r.Get("/ping", pingDBHandler.PingDBHandler(pinger))
	})

	server := &http.Server{
		Addr:    c.Address,
		Handler: r,
	}

	go func() {
		logger.Log.Info("Running server", zap.String("address", c.Address))
		if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("server error: ", zap.Error(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.Shutdown(shutdownCtx)

	auditService.Stop()
	shutdown()
	<-auditDoneCh
}
