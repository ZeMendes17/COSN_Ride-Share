package matching

import (
	"encoding/json"
	"matching-service/internal/db"
	"matching-service/internal/metrics"
	"matching-service/internal/ratelimit"
	"matching-service/internal/tracing"
	"matching-service/pkg/contracts"
	"matching-service/pkg/model"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func MakeHTTPHandler(s Service, repo db.Repository, m *metrics.Metrics, rl *ratelimit.RateLimiter) http.Handler {
	r := mux.NewRouter()

	// Middleware to record metrics
	metricsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m.InFlightRequests.Inc()
			defer m.InFlightRequests.Dec()

			start := time.Now()
			next.ServeHTTP(w, r)
			m.RequestLatencySeconds.Observe(time.Since(start).Seconds())
			m.RequestsTotal.Inc()
		})
	}

	// Middleware for rate limiting
	rateLimitMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use a default user ID if not specified
			userID := r.Header.Get("Authorization")
			if userID == "" {
				userID = "anonymous"
			}

			if err := rl.CanAcceptRequest(userID); err != nil {
				m.ErrorsTotal.Inc()
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			defer rl.ReleaseRequest()

			next.ServeHTTP(w, r)
		})
	}

	// Create a subrouter for /matches prefix to work with ALB ingress
	matches := r.PathPrefix("/matches").Subrouter()

	matches.HandleFunc("/health", makeHealthHandler(s)).Methods("GET")
	matches.HandleFunc("/ready", makeReadyHandler(s)).Methods("GET")

	// GET /metrics (Prometheus metrics)
	matches.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(rl.GetStats())
	}).Methods("GET")

	matches.Handle("/requests/{id}", metricsMiddleware(rateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from request
		tc := tracing.ExtractTraceContext(r)
		ctx := tracing.TraceContextToContext(r.Context(), tc)

		vars := mux.Vars(r)
		matches, err := s.GetMatchesForRequest(ctx, vars["id"])
		if err != nil {
			m.ErrorsTotal.Inc()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(matches)
	})))).Methods("GET")

	matches.HandleFunc("/{offerId}/requests/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		err := s.SelectMatch(r.Context(), vars["id"], vars["offerId"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}).Methods("POST")

	matches.HandleFunc("/{matchId}/requests/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		err := s.CancelMatch(r.Context(), vars["id"], vars["matchId"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Match cancelled"))
	}).Methods("DELETE")

	// --- MOCK TESTING ENDPOINTS ---

	// POST /test/offer (New JSON Format)
	matches.HandleFunc("/test/offer", func(w http.ResponseWriter, r *http.Request) {
		var event contracts.OfferTripAvailableEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		domainOffer := model.Offer{
			OfferID:          event.OfferID,
			DriverID:         event.DriverID,
			DriverName:       event.DriverName,
			Origin:           model.GeoLocation{Lat: event.OriginLat, Lon: event.OriginLon},
			Destination:      model.GeoLocation{Lat: event.DestinyLat, Lon: event.DestinyLon},
			AvailableSeats:   event.AvailableSeats,
			DepartureTimeMin: event.DepartureTimeMin,
			DepartureTimeMax: event.DepartureTimeMax,
			Waypoints:        event.Waypoints,
			Preferences: model.Preferences{
				Smoking: event.Preferences.Smoking,
				Pets:    event.Preferences.Pets,
				Music:   event.Preferences.Music,
			},
		}
		triggerReqIDs := []string{}
		if event.TriggerRequest != nil {
			for _, tr := range event.TriggerRequest {
				if tr.RequesterID != "" {
					triggerReqIDs = append(triggerReqIDs, tr.RequesterID)
				}
				if tr.PendingRequestIds != nil {
					triggerReqIDs = append(triggerReqIDs, tr.PendingRequestIds...)
				}
			}
		}

		s.ProcessOffer(r.Context(), domainOffer, triggerReqIDs)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Offer processed successfully"))
	}).Methods("POST")

	// POST /test/offer/update (New JSON Format)
	matches.HandleFunc("/test/offer/update", func(w http.ResponseWriter, r *http.Request) {
		var event contracts.OfferUpdateEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		domainOffer := model.Offer{
			OfferID:          event.OfferID,
			DriverID:         event.DriverID,
			DriverName:       event.DriverName,
			Origin:           model.GeoLocation{Lat: event.OriginLat, Lon: event.OriginLon},
			Destination:      model.GeoLocation{Lat: event.DestinyLat, Lon: event.DestinyLon},
			AvailableSeats:   event.AvailableSeats,
			DepartureTimeMin: event.DepartureTimeMin,
			DepartureTimeMax: event.DepartureTimeMax,
			Waypoints:        event.Waypoints,
		}

		s.ProcessOffer(r.Context(), domainOffer, nil)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Offer update processed"))
	}).Methods("POST")

	return corsMiddleware(r)
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
