package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Request latency histogram
	RequestLatencySeconds prometheus.Histogram

	// Total request counters
	RequestsTotal prometheus.Counter
	ErrorsTotal   prometheus.Counter

	// In-flight requests gauge
	InFlightRequests prometheus.Gauge

	// Matching success/failure counters
	MatchesCreated prometheus.Counter
	MatchesFailed  prometheus.Counter

	// Database operation metrics
	DBQueryDurationSeconds prometheus.Histogram
	DBErrorsTotal          prometheus.Counter

	// Inter-service call metrics
	InterServiceCallsTotal     prometheus.Counter
	InterServiceErrorsTotal    prometheus.Counter
	InterServiceLatencySeconds prometheus.Histogram

	// Circuit breaker state
	CircuitBreakerState prometheus.Gauge
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
		MatchesCreated: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "matches_created_total",
				Help: "Total number of matches created",
			},
		),
		MatchesFailed: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "matches_failed_total",
				Help: "Total number of match attempts that failed",
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
		InterServiceCallsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "inter_service_calls_total",
				Help: "Total number of inter-service calls",
			},
		),
		InterServiceErrorsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "inter_service_errors_total",
				Help: "Total number of inter-service call errors",
			},
		),
		InterServiceLatencySeconds: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "inter_service_call_duration_seconds",
				Help:    "Inter-service call latency in seconds",
				Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
		),
		CircuitBreakerState: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
		),
	}
}
