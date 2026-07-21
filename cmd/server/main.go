package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/gzip"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/memStorage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	c := config.NewConfigServer()
	flag.Parse()
	c.ApplyEnv()

	base := memStorage.NewMemStorage()

	err := logger.InitializeLogger("info")
	if err != nil {
		logger.Log.Warn("Failed to initialize logger", zap.Error(err))
	}

	if c.Restore {
		if err = base.RestoreFromFile(c.FileStoragePath); err != nil {
			logger.Log.Warn("Failed to restore data from file", zap.Error(err))
		}
	}

	var store storage.ServerStorage = base
	var shutdown = func() {}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if c.StoreInterval == 0 {
		syncStore := memStorage.NewSyncMemStorage(store, c.FileStoragePath)
		store = syncStore
		shutdown = syncStore.Close
	} else {
		done := make(chan struct{})
		go service.SaveToFile(ctx, c, store, done)
		shutdown = func() { <-done }
	}

	db, err := sql.Open("pgx", c.DB)
	if err != nil {
		logger.Log.Fatal("db connection error: ", zap.Error(err))
	}
	defer db.Close()

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
		r.Get("/ping", handler.PingDBHandler(db))
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

	shutdown()
}
