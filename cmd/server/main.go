package main

import (
	"flag"
	"net/http"

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

	err := http.ListenAndServe(*address, r)
	if err != nil {
		panic(err)
	}
}
