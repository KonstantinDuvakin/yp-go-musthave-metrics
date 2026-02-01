package main

import (
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage"
)

const pollInterval = 2 * time.Second
const reportInterval = 10 * time.Second

func main() {
	store := storage.NewAgentStorage()
	agent := New(store)

	agent.Run(pollInterval, reportInterval)
}
