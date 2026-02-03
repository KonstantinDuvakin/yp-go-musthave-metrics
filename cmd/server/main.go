package main

import (
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

func main() {
	store := storage.NewMemStorage()

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

	err := http.ListenAndServe(`:8080`, r)
	if err != nil {
		panic(err)
	}
}
