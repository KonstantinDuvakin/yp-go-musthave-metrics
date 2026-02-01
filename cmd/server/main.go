package main

import (
	"net/http"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

func main() {
	store := storage.NewMemStorage()

	mux := http.NewServeMux()
	mux.HandleFunc(`/update/`, handler.UpdateHandler(store))

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
