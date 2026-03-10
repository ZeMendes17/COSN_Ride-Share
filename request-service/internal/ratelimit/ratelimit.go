package ratelimit

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter manages per-user and global rate limiting
type RateLimiter struct {
	mu             sync.RWMutex
	globalLimiter  *rate.Limiter
	userLimiters   map[string]*rate.Limiter
	globalRPS      float64 // Requests per second
	perUserRPS     float64 // Requests per second per user
	maxConcurrency int     // Max in-flight requests
	inFlightCount  int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(globalRPS, perUserRPS float64, maxConcurrency int) *RateLimiter {
	return &RateLimiter{
		globalLimiter:  rate.NewLimiter(rate.Limit(globalRPS), int(globalRPS)*10),
		userLimiters:   make(map[string]*rate.Limiter),
		globalRPS:      globalRPS,
		perUserRPS:     perUserRPS,
		maxConcurrency: maxConcurrency,
	}
}

// AllowGlobal checks if a request is allowed globally
func (rl *RateLimiter) AllowGlobal(ctx time.Duration) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return rl.globalLimiter.AllowN(time.Now(), 1)
}

// AllowUser checks if a user is allowed to make a request
func (rl *RateLimiter) AllowUser(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get or create user limiter
	limiter, ok := rl.userLimiters[userID]
	if !ok {
		limiter = rate.NewLimiter(rate.Limit(rl.perUserRPS), int(rl.perUserRPS)*10)
		rl.userLimiters[userID] = limiter
	}

	return limiter.AllowN(time.Now(), 1)
}

// CanAcceptRequest checks both global and concurrency limits
func (rl *RateLimiter) CanAcceptRequest(userID string) error {
	// Check global limit
	if !rl.AllowGlobal(0) {
		return fmt.Errorf("global rate limit exceeded")
	}

	// Check per-user limit
	if !rl.AllowUser(userID) {
		return fmt.Errorf("user rate limit exceeded")
	}

	// Check concurrency limit
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.inFlightCount >= rl.maxConcurrency {
		return fmt.Errorf("server concurrency limit exceeded")
	}
	rl.inFlightCount++

	return nil
}

// ReleaseRequest decrements the in-flight counter
func (rl *RateLimiter) ReleaseRequest() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.inFlightCount > 0 {
		rl.inFlightCount--
	}
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"in_flight_requests": rl.inFlightCount,
		"max_concurrency":    rl.maxConcurrency,
		"active_users":       len(rl.userLimiters),
		"global_rps":         rl.globalRPS,
		"per_user_rps":       rl.perUserRPS,
	}
}
