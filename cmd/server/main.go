package main

import (
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/repository"
)

func main() {
	storage := &repository.MemStorage{
		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(`/update/`, handler.UpdateHandler(storage))

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
