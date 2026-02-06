package main

import (
	"flag"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

var (
	address   = flag.String("a", "localhost:8080", "The address to listen on for HTTP requests.")
	pollSec   = flag.Int("p", 2, "defined poll interval duration")
	reportSec = flag.Int("r", 10, "defined report interval duration")
)

func main() {
	flag.Parse()

	store := storage.NewAgentStorage()
	agent := New(store, *address)

	pollInterval := time.Duration(*pollSec) * time.Second
	reportInterval := time.Duration(*reportSec) * time.Second

	agent.Run(pollInterval, reportInterval)
}
