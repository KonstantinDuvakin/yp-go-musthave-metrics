// Команда agent периодически собирает метрики среды выполнения и системы и
// отправляет их на сервер сбора метрик. Параметры задаются флагами и
// переменными окружения (см. [config.AgentConfig]).
package main

import (
	"context"
	"crypto/rsa"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/agent/sender"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/build_info"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/crypto"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/agent_storage"
	"go.uber.org/zap"
)

var (
	buildVersion = buildinfo.NA
	buildDate    = buildinfo.NA
	buildCommit  = buildinfo.NA
)

func main() {
	buildinfo.PrintBuildInfo(os.Stdout, buildVersion, buildDate, buildCommit)

	c := config.NewConfigAgent()
	flag.Parse()
	c.ApplyEnv()

	err := logger.InitializeLogger("info")
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v", err)
	}

	store := agentstorage.NewAgentStorage()
	send := sender.NewSender(c.Address)
	agent := New(store, send)

	var pubKey *rsa.PublicKey
	if c.CryptoKey != "" {
		pubKey, err = crypto.ReadPublicKey(c.CryptoKey)
		if err != nil {
			logger.Log.Fatal("Can't get public key", zap.Error(err))
		}
	}

	pollInterval := time.Duration(c.PollSec) * time.Second
	reportInterval := time.Duration(c.ReportSec) * time.Second

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	agent.Run(ctx, pollInterval, reportInterval, c.Key, c.RateLimit, pubKey)
}
