// Команда server принимает метрики от агентов по HTTP, хранит их в памяти
// или в базе данных и отдаёт по запросу. Параметры задаются флагами и
// переменными окружения (см. [config.ServerConfig]).
package main

import (
	"context"
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
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
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/ipmw"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/middlewares/logger"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/metrics"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/send_to_audit"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/service/update_metrics"
	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/storage/server_storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	store, db, shutdown, err := serverstorage.NewStorage(ctx, c)
	if err != nil {
		logger.Log.Fatal("Failed to initialize storage", zap.Error(err))
	}

	var pinger pingdbhandler.Pinger
	if db != nil {
		pinger = db
	}

	var ipNet *net.IPNet
	if c.TrustedSubnet != "" {
		_, ipNet, err = net.ParseCIDR(c.TrustedSubnet)
		if err != nil {
			logger.Log.Fatal("Invalid trusted subnet", zap.Error(err))
		}
	}

	var privateKey *rsa.PrivateKey
	if c.CryptoKey != "" {
		privateKey, err = crypto.ReadPrivateKey(c.CryptoKey)
		if err != nil {
			logger.Log.Fatal("Can't get private key", zap.Error(err))
		}
	}

	auditService := sendtoaudit.NewAuditService(c.AuditFile, c.AuditURL)
	auditService.Start()

	batchService := updatemetrics.New(store, auditService)

	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Use(logger.RequestLogger)
		if ipNet != nil {
			r.Use(ipmw.Middleware(ipNet))
		}
		if privateKey != nil {
			r.Use(cryptomw.Middleware(privateKey))
		}
		r.Use(gzip.Middleware)
		r.Use(hash.Middleware(c.Key))

		r.Get("/", roothandler.RootHandler(store))
		r.Route("/update", func(r chi.Router) {
			r.Post("/", updatemetricjson.UpdateMetricJSON(store))
			r.Post(`/{type}/{name}/{value}`, updatehandler.UpdateHandler(store))
		})
		r.Route("/updates", func(r chi.Router) {
			r.Post("/", updatebatchmetrics.UpdateBatchMetrics(batchService))
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

	eg, gctx := errgroup.WithContext(ctx)

	if c.GRPCAddress != "" {
		eg.Go(func() error {
			metricsGRPC := metrics.NewMetrics(batchService)
			return metricsGRPC.Run(gctx, c.GRPCAddress, c.GRPCCert, c.GRPCKey, ipNet)
		})
	}

	eg.Go(func() error {
		logger.Log.Info("Running server", zap.String("address", c.Address))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		<-gctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return server.Shutdown(shutdownCtx)
	})

	err = eg.Wait()

	auditService.Stop()
	shutdown()

	if err != nil {
		logger.Log.Error("Server stopped with error: ", zap.Error(err))
		os.Exit(1)
	}
}
