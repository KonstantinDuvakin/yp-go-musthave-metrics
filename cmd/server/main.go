package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func main() {
	store := storage.NewMemStorage()

	c := config.NewConfigServer()

	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Get("/", handler.RootHandler(store))
		r.Route("/update", func(r chi.Router) {
			r.Post(`/{type}/{name}/{value}`, handler.UpdateHandler(store))
		})
		r.Route("/value", func(r chi.Router) {
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
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.Shutdown(shutdownCtx)
}
