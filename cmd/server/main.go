package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func main() {
	store := storage.NewMemStorage()

	address := flag.String("a", "localhost:8080", "set an address of a server")

	flag.Parse()

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
		Addr:    *address,
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
