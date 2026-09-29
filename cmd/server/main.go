// Команда server принимает метрики от агентов по HTTP, хранит их в памяти
// или в базе данных и отдаёт по запросу. Параметры задаются флагами и
// переменными окружения (см. [config.ServerConfig]).
package main

import (
	"context"
	"crypto/rsa"
	"errors"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/config"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/get_metric_handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/get_metric_json"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/ping_db_handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/root_handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/update_batch_metrics"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/update_handler"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/handlers/update_metric_json"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/build_info"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/helpers/crypto"
	cryptomw "github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/crypto"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/gzip"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/hash"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/send_to_audit"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

var (
	buildVersion = buildinfo.NA
	buildDate    = buildinfo.NA
	buildCommit  = buildinfo.NA
)

func main() {
	buildinfo.PrintBuildInfo(os.Stdout, buildVersion, buildDate, buildCommit)

	err := logger.InitializeLogger("info")
	if err != nil {
		log.Fatal("Failed to initialize logger: ", err)
	}

	c := config.NewConfigServer()

	if err = config.ParseFile(c); err != nil {
		logger.Log.Fatal("Failed parse configuration file", zap.Error(err))
	}

	flag.Parse()
	c.ApplyEnv()

	err = c.Validate()
	if err != nil {
		logger.Log.Fatal("Error validate configuration", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, db, shutdown, err := serverstorage.NewStorage(ctx, c)
	if err != nil {
		logger.Log.Fatal("Failed to initialize storage", zap.Error(err))
	}

	var pinger pingdbhandler.Pinger
	if db != nil {
		pinger = db
	}

	var privateKey *rsa.PrivateKey
	if c.CryptoKey != "" {
		privateKey, err = crypto.ReadPrivateKey(c.CryptoKey)
		if err != nil {
			logger.Log.Fatal("Can't get private key", zap.Error(err))
		}
	}

	auditDoneCh := make(chan struct{})
	auditService := sendtoaudit.NewAuditService(c.AuditFile, c.AuditURL)
	go func() {
		auditService.Start()
		close(auditDoneCh)
	}()

	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Use(logger.RequestLogger)
		r.Use(cryptomw.Middleware(privateKey))
		r.Use(gzip.Middleware)
		r.Use(hash.Middleware(c.Key))

		r.Get("/", roothandler.RootHandler(store))
		r.Route("/update", func(r chi.Router) {
			r.Post("/", updatemetricjson.UpdateMetricJSON(store))
			r.Post(`/{type}/{name}/{value}`, updatehandler.UpdateHandler(store))
		})
		r.Route("/updates", func(r chi.Router) {
			r.Post("/", updatebatchmetrics.UpdateBatchMetrics(store, auditService.SendEvent))
		})
		r.Route("/value", func(r chi.Router) {
			r.Post("/", getmetricjson.GetMetricJSON(store))
			r.Get(`/{type}/{name}`, getmetrichandler.GetMetricHandler(store))
		})
		r.Get("/ping", pingdbhandler.PingDBHandler(pinger))
	})

	r.Mount("/debug", middleware.Profiler())

	server := &http.Server{
		Addr:    c.Address,
		Handler: r,
	}

	go func() {
		logger.Log.Info("Running server", zap.String("address", c.Address))
		if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("server error: ", zap.Error(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.Shutdown(shutdownCtx)

	auditService.Stop()
	shutdown()
	<-auditDoneCh
}
