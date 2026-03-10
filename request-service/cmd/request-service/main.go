package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"request-service/internal/config"
	"request-service/internal/db"
	"request-service/internal/metrics"
	"request-service/internal/ratelimit"
	"request-service/internal/request"
	"request-service/internal/sns"

	kitlog "github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var logger kitlog.Logger
	{
		logger = kitlog.NewLogfmtLogger(os.Stderr)
		logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
		logger = kitlog.With(logger, "caller", kitlog.DefaultCaller)
	}

	cfg := config.LoadConfig()

	// Build PostgreSQL connection string
	connStr := cfg.GetDBConnString()
	_ = logger.Log("msg", "Connecting to PostgreSQL", "host", cfg.DBHost)

	repo, err := db.NewPostgresRepository(connStr)
	if err != nil {
		_ = logger.Log("error", "failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer func() { _ = repo.Close() }()
	_ = logger.Log("msg", "Connected to PostgreSQL database")

	// Create SNS producer
	var producer sns.Producer
	if cfg.ExternalAWSAccessKey != "" && cfg.SNSRequestCreatedARN != "" {
		snsProducer, err := sns.NewSNSProducer(
			cfg.AWSRegion,
			cfg.ExternalAWSAccessKey,
			cfg.ExternalAWSSecretKey,
			cfg.SNSRequestCreatedARN,
			cfg.SNSRequestCancelledARN,
			cfg.SNSLoggingARN,
		)
		if err != nil {
			_ = logger.Log("error", "failed to create SNS producer", "err", err)
			os.Exit(1)
		}
		producer = snsProducer
		_ = logger.Log("msg", "SNS producer initialized",
			"requestCreatedARN", cfg.SNSRequestCreatedARN,
			"requestCancelledARN", cfg.SNSRequestCancelledARN)
	} else {
		producer = sns.NewNoOpProducer()
		_ = logger.Log("msg", "Using NoOp producer (SNS credentials not provided)")
	}
	defer repo.Close()
	logger.Log("msg", "Connected to Postgres database")

	// Initialize rate limiter (50 RPS global, 5 RPS per user, 500 max concurrent)
	rl := ratelimit.NewRateLimiter(50.0, 5.0, 500)
	logger.Log("msg", "Initialized rate limiter", "global_rps", 50, "per_user_rps", 5)

	// Initialize metrics
	m := metrics.NewMetrics()
	logger.Log("msg", "Initialized Prometheus metrics")

	svc := request.NewBasicService(repo, producer)

	// Create endpoints
	createReq := request.MakeCreateRequestEndpoint(svc)
	getReq := request.MakeGetRequestsEndpoint(svc)
	getAllPending := request.MakeGetAllPendingRequestsEndpoint(svc)
	cancelReq := request.MakeCancelRequestEndpoint(svc)
	patchReq := request.MakePatchRequestEndpoint(svc)

	authMiddleware := request.AuthMiddleware()

	createReq = authMiddleware(createReq)
	getReq = authMiddleware(getReq)
	getAllPending = authMiddleware(getAllPending)
	cancelReq = authMiddleware(cancelReq)
	patchReq = authMiddleware(patchReq)

	endpoints := request.Endpoints{
		CreateRequestEndpoint:         createReq,
		GetRequestsEndpoint:           getReq,        // Public/Authenticated implicitly by logic if needed
		GetAllPendingRequestsEndpoint: getAllPending, // Public/Admin
		CancelRequestEndpoint:         cancelReq,
		PatchRequestEndpoint:          patchReq,
	}

	httpHandler := request.MakeHTTPHandler(svc, endpoints, logger, repo, m, rl)

	errs := make(chan error)

	// Setup graceful shutdown on OS signal
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- (<-c).(error)
	}()

	// HTTP server with timeout
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Prometheus metrics endpoint on separate port (9091)
	go func() {
		promSrv := &http.Server{
			Addr:    ":9091",
			Handler: promhttp.Handler(),
		}
		logger.Log("transport", "Prometheus", "addr", ":9091")
		promSrv.ListenAndServe()
	}()

	go func() {
		logger.Log("transport", "HTTP", "addr", cfg.HTTPAddr)
		errs <- srv.ListenAndServe()
	}()

	// Wait for signal, then shutdown gracefully
	logger.Log("exit", <-errs)
	logger.Log("msg", "Initiating graceful shutdown")

	// Give server 10s to shutdown gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log("error", "shutdown error", "err", err)
	}

	logger.Log("msg", "Shutdown complete")
}
