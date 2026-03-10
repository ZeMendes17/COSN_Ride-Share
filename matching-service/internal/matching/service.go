package matching

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"matching-service/internal/db"
	"matching-service/internal/metrics"
	"matching-service/internal/resilience"
	"matching-service/internal/tracing"
	"matching-service/pkg/contracts"
	"matching-service/pkg/model"
)

type Producer interface {
	SendMatchCreated(ctx context.Context, event contracts.MatchCreatedEvent) error
	SendMatchCancelled(ctx context.Context, event contracts.MatchCancelledEvent) error
	SendLog(ctx context.Context, event contracts.LogEvent) error
}

type Service interface {
	GetMatchesForRequest(ctx context.Context, requestID string) ([]model.Match, error)
	SelectMatch(ctx context.Context, requestID, offerID string) error
	CancelMatch(ctx context.Context, requestID, matchID string) error
	ProcessOffer(ctx context.Context, offer model.Offer, pendingRequestIDs []string) error
	StartAnnealing(ctx context.Context)
	HealthCheck() error
}

type BasicService struct {
	repo           db.Repository
	algorithm      AlgorithmService
	producer       Producer
	requestSvcURL  string
	logger         log.Logger
	httpClient     *http.Client
	metrics        *metrics.Metrics
	circuitBreaker *resilience.CircuitBreaker
	retryConfig    resilience.RetryConfig
}

func NewBasicService(repo db.Repository, prod Producer, requestSvcURL string, logger log.Logger, m *metrics.Metrics) Service {
	return &BasicService{
		repo:           repo,
		algorithm:      AlgorithmService{},
		producer:       prod,
		requestSvcURL:  requestSvcURL,
		logger:         logger,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		metrics:        m,
		circuitBreaker: resilience.NewCircuitBreaker(5, 2, 30*time.Second),
		retryConfig:    resilience.DefaultRetryConfig(),
	}
}

// StartAnnealing runs in the background
func (s *BasicService) StartAnnealing(ctx context.Context) {
	// Run checks frequently (every 30s) to catch requests that just crossed a time threshold
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	s.logger.Log("msg", "Annealing service started", "interval", "30s")

	for {
		select {
		case <-ctx.Done():
			s.logger.Log("msg", "Annealing service shutting down")
			return
		case <-ticker.C:
			s.runAnnealingCycle(ctx)
		}
	}
}

func (s *BasicService) runAnnealingCycle(ctx context.Context) {
	// 1. Get Pending Requests
	reqs, err := s.fetchAllPendingRequests(ctx)
	if err != nil {
		s.logger.Log("msg", "Annealing: Failed to fetch requests", "err", err)
		return
	}

	if len(reqs) == 0 {
		return // Silent return if nothing to do
	}

	// 2. Get All Offers
	offers, err := s.repo.GetAllOffers(ctx)
	if err != nil {
		s.logger.Log("msg", "Annealing: Failed to fetch offers", "err", err)
		return
	}
	if len(offers) == 0 {
		return // Silent return if no offers
	}

	for _, req := range reqs {
		// 3. Calculate Dynamic Radius
		// Base: 1km. Add 500m (0.5km) for every 2 minutes waiting.
		minutesWaiting := time.Since(req.CreatedAt).Minutes()
		expansionSteps := int(minutesWaiting / 2.0) // Floor division
		dynamicRadius := DefaultBaseDistance + (float64(expansionSteps) * AnnealingExpansion)

		// Only log if we are actually expanding significantly (e.g. > 1km) to reduce noise
		if dynamicRadius > DefaultBaseDistance {
			// Optional: Log only periodically or at debug level
			// log.Printf("Annealing Req %s: Waiting %.1f min -> Radius %.1f km", req.ID, minutesWaiting, dynamicRadius)
		}

		for _, offer := range offers {
			match, ok := s.algorithm.CalculateMatch(req, offer, dynamicRadius)
			if ok {
				// Check for existing match to prevent duplicates
				existing, _ := s.repo.GetMatchesByRequestID(ctx, req.ID)
				alreadyMatched := false
				for _, m := range existing {
					if m.OfferID == offer.OfferID {
						alreadyMatched = true
						break
					}
				}

				if !alreadyMatched {
					s.logger.Log("msg", "Annealing match found", "requestID", req.ID, "offerID", offer.OfferID, "radiusKm", fmt.Sprintf("%.1f", dynamicRadius))
					match.MatchID = fmt.Sprintf("match-%d-anneal", time.Now().UnixNano())
					s.repo.SaveMatch(ctx, match)

					event := contracts.MatchCreatedEvent{
						MatchID: match.MatchID, RequestID: match.RequestID, OfferID: match.OfferID,
						DriverID: match.DriverID, PassengerID: match.PassengerID,
						EstimatedPickupTime: match.EstimatedPickupTime, Timestamp: time.Now(),
					}
					s.producer.SendMatchCreated(ctx, event)
					s.producer.SendLog(ctx, contracts.LogEvent{
						ServiceID: "matching-service",
						Topic:     "match.created.anneal",
						Message:   fmt.Sprintf("Match created via annealing: MatchID=%s, RequestID=%s, OfferID=%s, DriverID=%s", match.MatchID, match.RequestID, match.OfferID, match.DriverID),
					})
				}
			}
		}
	}
}

func (s *BasicService) ProcessOffer(ctx context.Context, offer model.Offer, pendingRequestIDs []string) error {
	// Save Offer for future annealing
	if err := s.repo.SaveOffer(ctx, offer); err != nil {
		s.logger.Log("msg", "Failed to save offer", "offerID", offer.OfferID, "err", err)
	}

	s.logger.Log("msg", "Processing offer", "offerID", offer.OfferID, "pendingRequestIDs", fmt.Sprintf("%v", pendingRequestIDs))

	if pendingRequestIDs == nil {
		if err := s.repo.ClearPendingMatchesForOffer(ctx, offer.OfferID); err != nil {
			s.logger.Log("msg", "Failed to clear old matches", "offerID", offer.OfferID, "err", err)
		}
	}
	var requests []model.Request
	if len(pendingRequestIDs) > 0 {
		for _, id := range pendingRequestIDs {
			req, err := s.fetchRequest(ctx, id)
			if err == nil {
				requests = append(requests, req)
			}
		}
	} else {
		allReqs, err := s.fetchAllPendingRequests(ctx)
		if err == nil {
			requests = allReqs
		}
	}

	for _, req := range requests {
		// Initial match uses standard base radius (1km)
		match, ok := s.algorithm.CalculateMatch(req, offer, DefaultBaseDistance)
		if !ok {
			s.metrics.MatchesFailed.Inc()
			continue
		}

		match.MatchID = fmt.Sprintf("match-%d-%s", time.Now().UnixNano(), req.ID)
		if err := s.repo.SaveMatch(ctx, match); err != nil {
			s.metrics.MatchesFailed.Inc()
			continue
		}
		s.metrics.MatchesCreated.Inc()
		s.producer.SendLog(ctx, contracts.LogEvent{
			ServiceID: "matching-service",
			Topic:     "match.created",
			Message:   fmt.Sprintf("Match created: MatchID=%s, RequestID=%s, OfferID=%s, DriverID=%s", match.MatchID, req.ID, offer.OfferID, offer.DriverID),
		})
	}
	return nil
}

func (s *BasicService) GetMatchesForRequest(ctx context.Context, requestID string) ([]model.Match, error) {
	return s.repo.GetMatchesByRequestID(ctx, requestID)
}
func (s *BasicService) SelectMatch(ctx context.Context, requestID, offerID string) error {
	s.logger.Log("msg", "Selecting match", "requestID", requestID, "offerID", offerID)
	s.producer.SendLog(ctx, contracts.LogEvent{
		ServiceID: "matching-service",
		Topic:     "match.selected",
		Message:   fmt.Sprintf("Match selected: RequestID=%s, OfferID=%s", requestID, offerID),
	})

	match, err := s.repo.GetMatchByOfferIDRequestID(ctx, offerID, requestID)
	if err != nil {
		return err
	}

	if err := s.updateRequestStatus(ctx, requestID, "Completed"); err != nil {
		return err
	}

	event := contracts.MatchCreatedEvent{
		MatchID: match.MatchID, RequestID: match.RequestID, OfferID: match.OfferID,
		DriverID: match.DriverID, PassengerID: match.PassengerID,
		EstimatedPickupTime: match.EstimatedPickupTime, Timestamp: time.Now(),
	}

	if err := s.producer.SendMatchCreated(ctx, event); err != nil {
		return err
	}
	
	if err := s.repo.UpdateMatchStatus(ctx, match.MatchID, "Completed"); err != nil {
		return err
	}
	return nil
}
func (s *BasicService) CancelMatch(ctx context.Context, requestID, matchID string) error {
	match, err := s.repo.GetMatchByID(ctx, matchID)
	if err != nil {
		return err
	}
	if match.RequestID != requestID {
		return fmt.Errorf("match does not belong to request")
	}
	if err := s.repo.UpdateMatchStatus(ctx, matchID, "Cancelled"); err != nil {
		return err
	}
	event := contracts.MatchCancelledEvent{
		MatchID: match.MatchID, RequestID: match.RequestID, OfferID: match.OfferID,
		Reason: "user_initiated_cancel", Timestamp: time.Now(),
	}
	if err := s.producer.SendMatchCancelled(ctx, event); err != nil {
		return err
	}
	if err := s.updateRequestStatus(ctx, requestID, "Pending"); err != nil {
		return err
	}
	return nil
}
func (s *BasicService) fetchAllPendingRequests(ctx context.Context) ([]model.Request, error) {
	// Check circuit breaker first
	if err := s.circuitBreaker.CanExecute(); err != nil {
		s.logger.Log("msg", "Circuit breaker open", "err", err)
		s.metrics.InterServiceErrorsTotal.Inc()
		return nil, err
	}

	// Extract or generate trace context
	tc, ok := tracing.TraceContextFromContext(ctx)
	if !ok {
		tc = tracing.TraceContext{
			TraceID: tracing.GenerateTraceID(),
			SpanID:  tracing.GenerateSpanID(),
		}
	}

	var requests []model.Request
	start := time.Now()

	// Retry with exponential backoff
	err := resilience.Retry(ctx, s.retryConfig, func(retryCtx context.Context) error {
		url := fmt.Sprintf("%s/requests/pending", s.requestSvcURL)
		req, err := http.NewRequestWithContext(retryCtx, "GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "system")

		// Inject trace context
		tracing.InjectTraceContext(req, tc)

		s.metrics.InterServiceCallsTotal.Inc()
		resp, err := s.httpClient.Do(req)
		if err != nil {
			s.circuitBreaker.RecordFailure()
			s.logger.Log("msg", "Failed to fetch pending requests", "url", url, "traceID", tc.TraceID, "err", err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			s.circuitBreaker.RecordFailure()
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		if err := json.NewDecoder(resp.Body).Decode(&requests); err != nil {
			s.circuitBreaker.RecordFailure()
			s.logger.Log("msg", "Failed to decode pending requests", "traceID", tc.TraceID, "err", err)
			return err
		}

		s.circuitBreaker.RecordSuccess()
		return nil
	})

	if err != nil {
		s.metrics.InterServiceErrorsTotal.Inc()
		return nil, err
	}

	s.metrics.InterServiceLatencySeconds.Observe(time.Since(start).Seconds())
	return requests, nil
}
func (s *BasicService) fetchRequest(ctx context.Context, id string) (model.Request, error) {
	requests, err := s.fetchAllPendingRequests(ctx)
	if err != nil {
		return model.Request{}, err
	}
	for _, r := range requests {
		if r.ID == id {
			return r, nil
		}
	}
	return model.Request{}, fmt.Errorf("request %s not found in pending list", id)
}
func (s *BasicService) updateRequestStatus(ctx context.Context, id, status string) error {
	// Check circuit breaker
	if err := s.circuitBreaker.CanExecute(); err != nil {
		s.logger.Log("msg", "Circuit breaker open", "err", err)
		s.metrics.InterServiceErrorsTotal.Inc()
		return err
	}

	// Extract or generate trace context
	tc, ok := tracing.TraceContextFromContext(ctx)
	if !ok {
		tc = tracing.TraceContext{
			TraceID: tracing.GenerateTraceID(),
			SpanID:  tracing.GenerateSpanID(),
		}
	}

	start := time.Now()

	// Retry with exponential backoff
	err := resilience.Retry(ctx, s.retryConfig, func(retryCtx context.Context) error {
		url := fmt.Sprintf("%s/requests/%s", s.requestSvcURL, id)
		payload := map[string]interface{}{"status": status}
		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(retryCtx, "PATCH", url, bytes.NewBuffer(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "system")

		// Inject trace context
		tracing.InjectTraceContext(req, tc)

		s.metrics.InterServiceCallsTotal.Inc()
		resp, err := s.httpClient.Do(req)
		if err != nil {
			s.circuitBreaker.RecordFailure()
			s.logger.Log("msg", "Failed to update request status", "requestID", id, "traceID", tc.TraceID, "err", err)
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			s.circuitBreaker.RecordFailure()
			s.logger.Log("msg", "Failed to update request status", "requestID", id, "statusCode", resp.StatusCode, "traceID", tc.TraceID)
			return fmt.Errorf("failed to update status, code: %d", resp.StatusCode)
		}

		s.circuitBreaker.RecordSuccess()
		return nil
	})

	if err != nil {
		s.metrics.InterServiceErrorsTotal.Inc()
		return err
	}

	s.metrics.InterServiceLatencySeconds.Observe(time.Since(start).Seconds())
	return nil
}

func (s *BasicService) HealthCheck() error {
	return s.repo.Ping()
}
