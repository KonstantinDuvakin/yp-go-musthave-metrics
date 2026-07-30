package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/agentStorage"
)

func main() {
	c := config.NewConfigAgent()
	flag.Parse()
	c.ApplyEnv()

	store := agentStorage.NewAgentStorage()
	send := sender.NewSender(c.Address)
	agent := New(store, send)

	pollInterval := time.Duration(c.PollSec) * time.Second
	reportInterval := time.Duration(c.ReportSec) * time.Second

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	agent.Run(ctx, pollInterval, reportInterval, c.Key)
}
