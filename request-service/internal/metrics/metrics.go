package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for request-service
type Metrics struct {
	// Request latency histogram
	RequestLatencySeconds prometheus.Histogram

	// Total request counters
	RequestsTotal prometheus.Counter
	ErrorsTotal   prometheus.Counter

	// In-flight requests gauge
	InFlightRequests prometheus.Gauge

	// Request operation counters
	RequestsCreated   prometheus.Counter
	RequestsCancelled prometheus.Counter
	RequestsUpdated   prometheus.Counter

	// Database operation metrics
	DBQueryDurationSeconds prometheus.Histogram
	DBErrorsTotal          prometheus.Counter

	// Rate limiter state
	RateLimitExceededTotal prometheus.Counter
}

// NewMetrics creates and registers all metrics
func NewMetrics() *Metrics {
	return &Metrics{
		RequestLatencySeconds: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
		),
		RequestsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
		),
		ErrorsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "http_errors_total",
				Help: "Total number of HTTP errors",
			},
		),
		InFlightRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of in-flight HTTP requests",
			},
		),
		RequestsCreated: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "requests_created_total",
				Help: "Total number of requests created",
			},
		),
		RequestsCancelled: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "requests_cancelled_total",
				Help: "Total number of requests cancelled",
			},
		),
		RequestsUpdated: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "requests_updated_total",
				Help: "Total number of requests updated",
			},
		),
		DBQueryDurationSeconds: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
		),
		DBErrorsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "db_errors_total",
				Help: "Total number of database errors",
			},
		),
		RateLimitExceededTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "rate_limit_exceeded_total",
				Help: "Total number of rate limit exceeded errors",
			},
		),
	}
}
