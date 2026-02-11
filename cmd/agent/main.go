package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
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
	send := sender.NewSender(*address)
	agent := New(store, send)

	pollInterval := time.Duration(*pollSec) * time.Second
	reportInterval := time.Duration(*reportSec) * time.Second

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	agent.Run(ctx, pollInterval, reportInterval)
}
