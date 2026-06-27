package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/gzip"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	store := storage.NewMemStorage()

	c := config.NewConfigServer()

	err := logger.InitializeLogger("info")
	if err != nil {
		logger.Log.Warn("Failed to initialize logger", zap.Error(err))
	}

	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Use(logger.RequestLogger)
		r.Use(gzip.Middleware)

		r.Get("/", handler.RootHandler(store))
		r.Route("/update", func(r chi.Router) {
			r.Post("/", handler.UpdateMetricJson(store))
			r.Post(`/{type}/{name}/{value}`, handler.UpdateHandler(store))
		})
		r.Route("/value", func(r chi.Router) {
			r.Post("/", handler.GetMetricJson(store))
			r.Get(`/{type}/{name}`, handler.GetMetricHandler(store))
		})
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:    c.Address,
		Handler: r,
	}

	go func() {
		logger.Log.Info("Running server", zap.String("address", c.Address))
		if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("server error: %v", zap.Error(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.Shutdown(shutdownCtx)
}
