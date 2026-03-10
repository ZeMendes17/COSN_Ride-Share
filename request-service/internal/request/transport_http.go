package request

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"request-service/internal/db"
	"request-service/internal/metrics"
	"request-service/internal/ratelimit"
	"request-service/internal/tracing"
	"request-service/pkg/model"
	"strings"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
)

var (
	ErrBadRouting = errors.New("inconsistent mapping between route and handler")
)

func MakeHTTPHandler(s Service, e Endpoints, logger log.Logger, repo db.Repository, m *metrics.Metrics, rl *ratelimit.RateLimiter) http.Handler {
	r := mux.NewRouter()
	r.StrictSlash(true)
	
	// Middleware for extracting trace context and rate limiting
	extractTraceAndRateLimit := func(ctx context.Context, r *http.Request) context.Context {
		// Extract trace context
		tc := tracing.ExtractTraceContext(r)
		ctx = tracing.TraceContextToContext(ctx, tc)

		// Extract user ID and apply rate limiting
		userID := r.Header.Get("Authorization")
		if userID == "" {
			userID = "anonymous"
		}

		if err := rl.CanAcceptRequest(userID); err != nil {
			// Will be handled in the error encoder
			// record metric for rate limit exceeded
			if m != nil {
				m.RateLimitExceededTotal.Inc()
			}
			return context.WithValue(ctx, "rate_limit_error", err)
		}

		// Store user ID in context for deferred release
		return context.WithValue(ctx, "request_user_id", userID)
	}

	// Create a subrouter for /requests prefix to work with ALB ingress
	requests := r.PathPrefix("/requests").Subrouter()

	options := []httptransport.ServerOption{
		httptransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		httptransport.ServerErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) {
			// Release rate limit if a user ID was stored in the context
			if uid, ok := ctx.Value("request_user_id").(string); ok && uid != "" {
				rl.ReleaseRequest()
			}
			encodeError(ctx, err, w)
		}),
		httptransport.ServerBefore(func(ctx context.Context, r *http.Request) context.Context {
			// First extract the token/user ID from Authorization header into context
			ctx = extractTokenFromHeader(ctx, r)
			// Then extract trace context and apply rate limiting
			return extractTraceAndRateLimit(ctx, r)
		}),
	}

	// Request metrics middleware (declared here so it can be used when registering handlers)
	metricsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m != nil {
				m.InFlightRequests.Inc()
				defer m.InFlightRequests.Dec()
			}
			start := time.Now()
			next.ServeHTTP(w, r)
			if m != nil {
				m.RequestLatencySeconds.Observe(time.Since(start).Seconds())
				m.RequestsTotal.Inc()
			}
		})
	}

	// GET /health (liveness probe)
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")

	// GET /ready (readiness probe - checks DB connectivity)
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Simple check: try a no-op query
		_, err := repo.GetAll(r.Context(), "", "")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not_ready", "error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}).Methods("GET")

	// Health check endpoints
	requests.HandleFunc("/health", makeHealthHandler(s)).Methods("GET")
	requests.HandleFunc("/ready", makeReadyHandler(s)).Methods("GET")

	// POST /requests
	requests.Handle("/", metricsMiddleware(httptransport.NewServer(
		e.CreateRequestEndpoint,
		decodeCreateRequest,
		encodeResponseCreated,
		options...,
	))).Methods("POST")

	// GET /requests/pending
	requests.Handle("/pending", metricsMiddleware(httptransport.NewServer(
		e.GetAllPendingRequestsEndpoint,
		decodeGetAllPendingRequests,
		encodeResponsePendingList,
		options...,
	))).Methods("GET")

	// GET /requests
	requests.Handle("/", metricsMiddleware(httptransport.NewServer(
		e.GetRequestsEndpoint,
		decodeGetRequests,
		encodeResponseJSON,
		options...,
	))).Methods("GET")

	// DELETE /requests/{id}
	requests.Handle("/{id}", metricsMiddleware(httptransport.NewServer(
		e.CancelRequestEndpoint,
		decodeCancelRequest,
		encodeResponse,
		options...,
	))).Methods("DELETE")

	// PATCH /requests/{id}
	requests.Handle("/{id}", metricsMiddleware(httptransport.NewServer(
		e.PatchRequestEndpoint,
		decodePatchRequest,
		encodeResponse,
		options...,
	))).Methods("PATCH")

	return corsMiddleware(r)
}

type Endpoints struct {
	CreateRequestEndpoint         endpoint.Endpoint
	GetRequestsEndpoint           endpoint.Endpoint
	GetAllPendingRequestsEndpoint endpoint.Endpoint
	CancelRequestEndpoint         endpoint.Endpoint
	PatchRequestEndpoint          endpoint.Endpoint
}

func extractTokenFromHeader(ctx context.Context, r *http.Request) context.Context {
	// Expecting header "Authorization: passenger-123"
	// In production, this would be "Authorization: Bearer <jwt>"
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Simple clean up if user sends "Bearer passenger-123"
		token := strings.TrimPrefix(authHeader, "Bearer ")
		return context.WithValue(ctx, ContextKeyUserID, token)
	}
	return ctx
}

// --- Decoders ---

func decodeCreateRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req CreateRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return req, nil
}

func decodeGetRequests(_ context.Context, r *http.Request) (interface{}, error) {
	status := r.URL.Query().Get("status")
	return GetRequestsRequest{Status: status}, nil
}

func decodeGetAllPendingRequests(_ context.Context, r *http.Request) (interface{}, error) {
	return GetAllPendingRequestsRequest{}, nil
}

func decodeCancelRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return nil, ErrBadRouting
	}
	return CancelRequestRequest{ID: id}, nil
}

func decodePatchRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return nil, ErrBadRouting
	}
	var req PatchRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	req.ID = id
	return req, nil
}

// --- Encoders ---

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(context.Background(), e.error(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

func encodeResponseCreated(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.WriteHeader(http.StatusCreated)
	return encodeResponse(ctx, w, response)
}

func encodeResponseJSON(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(GetRequestsResponse)
	if resp.Error != "" {
		encodeError(context.Background(), errors.New(resp.Error), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if resp.Requests == nil {
		return json.NewEncoder(w).Encode([]model.CarRequest{})
	}
	return json.NewEncoder(w).Encode(resp.Requests)
}

func encodeResponsePendingList(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(GetAllPendingRequestsResponse)
	if resp.Error != "" {
		encodeError(context.Background(), errors.New(resp.Error), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if resp.Requests == nil {
		return json.NewEncoder(w).Encode([]model.CarRequest{})
	}
	return json.NewEncoder(w).Encode(resp.Requests)
}

type errorer interface {
	error() error
}

func (r CreateRequestResponse) error() error         { return parseError(r.Error) }
func (r GetRequestsResponse) error() error           { return parseError(r.Error) }
func (r GetAllPendingRequestsResponse) error() error { return parseError(r.Error) }
func (r CancelRequestResponse) error() error         { return parseError(r.Error) }
func (r PatchRequestResponse) error() error          { return parseError(r.Error) }

func parseError(err string) error {
	if err == "" {
		return nil
	}
	switch err {
	case model.ErrInvalidRequest.Error():
		return model.ErrInvalidRequest
	case model.ErrNotFound.Error():
		return model.ErrNotFound
	case model.ErrNotModified.Error():
		return model.ErrNotModified
	case model.ErrRequestFinalized.Error():
		return model.ErrRequestFinalized
	case ErrUnauthorized.Error():
		return ErrUnauthorized
	case ErrForbidden.Error():
		return ErrForbidden
	default:
		return errors.New(err)
	}
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err {
	case model.ErrInvalidRequest:
		w.WriteHeader(http.StatusBadRequest) // 400
	case model.ErrNotFound:
		w.WriteHeader(http.StatusNotFound) // 404
	case model.ErrNotModified, model.ErrRequestFinalized:
		w.WriteHeader(http.StatusConflict) // 409
	case ErrUnauthorized:
		w.WriteHeader(http.StatusUnauthorized) // 401
	case ErrForbidden:
		w.WriteHeader(http.StatusForbidden) // 403
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func makeHealthHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.HealthCheck(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
	}
}

func makeReadyHandler(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.HealthCheck(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "not ready",
				"error":  err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	}
}
