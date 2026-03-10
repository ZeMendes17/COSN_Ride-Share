package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"matching-service/internal/config"
	"matching-service/internal/db"
	"matching-service/internal/matching"
	"matching-service/internal/metrics"
	"matching-service/internal/ratelimit"
	"matching-service/internal/sns"
	"matching-service/internal/sqs"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	cfg := config.LoadConfig()

	// Build PostgreSQL connection string
	connStr := cfg.GetDBConnString()
	logger.Log("msg", "Connecting to PostgreSQL", "host", cfg.DBHost)

	repo, err := db.NewPostgresRepository(connStr)
	if err != nil {
		logger.Log("error", "failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer func() { _ = repo.Close() }()
	logger.Log("msg", "Connected to PostgreSQL database")

	// Create SNS producer
	var producer matching.Producer
	if cfg.SNSAccessKey != "" && cfg.SNSMatchCreatedARN != "" {
		snsProducer, err := sns.NewSNSProducer(
			cfg.AWSRegion,
			cfg.SNSAccessKey,
			cfg.SNSSecretKey,
			cfg.SNSMatchCreatedARN,
			cfg.SNSMatchCancelledARN,
			cfg.SNSLoggingARN,
		)
		if err != nil {
			logger.Log("error", "failed to create SNS producer", "err", err)
			os.Exit(1)
		}
		producer = snsProducer
		logger.Log("msg", "SNS producer initialized",
			"matchCreatedARN", cfg.SNSMatchCreatedARN,
			"matchCancelledARN", cfg.SNSMatchCancelledARN)
	} else {
		producer = sns.NewNoOpProducer()
		logger.Log("msg", "Using NoOp producer (SNS credentials not provided)")
	}

	// Initialize metrics
	m := metrics.NewMetrics()
	logger.Log("msg", "Initialized Prometheus metrics")

	// Initialize rate limiter (100 RPS global, 10 RPS per user, 1000 max concurrent)
	rl := ratelimit.NewRateLimiter(100.0, 10.0, 1000)
	logger.Log("msg", "Initialized rate limiter", "global_rps", 100, "per_user_rps", 10)

	// Create matching service
	svc := matching.NewBasicService(repo, producer, cfg.RequestServiceURL, logger, m)

	// Create SQS consumer (needs the service to process messages)
	var consumer sqs.Consumer
	if cfg.SQSTripAvailableAccessKey != "" && cfg.SQSUpdateOfferAccessKey != "" {
		sqsConsumer, err := sqs.NewSQSConsumer(
			cfg.AWSRegion,
			cfg.SQSTripAvailableAccessKey,
			cfg.SQSTripAvailableSecretKey,
			cfg.SQSTripAvailableURL,
			cfg.SQSUpdateOfferAccessKey,
			cfg.SQSUpdateOfferSecretKey,
			cfg.SQSUpdateOfferURL,
			svc,
			logger,
		)
		if err != nil {
			logger.Log("error", "failed to create SQS consumer", "err", err)
			os.Exit(1)
		}
		consumer = sqsConsumer
		logger.Log("msg", "SQS consumer initialized with separate credentials for each queue")

		// Start consuming messages
		ctx := context.Background()
		if err := consumer.Start(ctx); err != nil {
			logger.Log("error", "failed to start SQS consumer", "err", err)
			os.Exit(1)
		}
	} else {
		consumer = sqs.NewNoOpConsumer()
		logger.Log("msg", "Using NoOp consumer (SQS credentials not provided)")
	}

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start annealing service in background
	go svc.StartAnnealing(ctx)
	logger.Log("msg", "Started annealing service")

	// Setup HTTP handler
	httpHandler := matching.MakeHTTPHandler(svc, repo, m, rl)

	// Setup graceful shutdown
	errs := make(chan error)

	// Setup graceful shutdown on OS signal
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	// HTTP server with timeout
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Prometheus metrics endpoint on separate port
	go func() {
		promSrv := &http.Server{
			Addr:    ":9090",
			Handler: promhttp.Handler(),
		}
		logger.Log("transport", "Prometheus", "addr", ":9090")
		_ = promSrv.ListenAndServe()
	}()

	go func() {
		logger.Log("transport", "HTTP", "addr", srv.Addr)
		errs <- srv.ListenAndServe()
	}()

	// Wait for signal, then shutdown gracefully
	logger.Log("exit", <-errs)
	logger.Log("msg", "Initiating graceful shutdown")

	// Cancel background tasks
	cancel()

	// Stop consumer if running
	if consumer != nil {
		_ = consumer.Stop()
	}

	// Give server 10s to shutdown gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log("error", "shutdown error", "err", err)
	}

	logger.Log("msg", "Shutdown complete")
}
