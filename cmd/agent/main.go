package main

import (
	"flag"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

var (
	address        = flag.String("a", "localhost:8080", "The address to listen on for HTTP requests.")
	pollInterval   = flag.Duration("p", 2*time.Second, "defined poll interval duration")
	reportInterval = flag.Duration("r", 10*time.Second, "defined report interval duration")
)

func main() {
	flag.Parse()

	store := storage.NewAgentStorage()
	agent := New(store, address)

	agent.Run(*pollInterval, *reportInterval)
}
