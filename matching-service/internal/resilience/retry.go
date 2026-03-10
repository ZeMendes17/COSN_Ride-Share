package resilience

import (
	"context"
	"math/rand"
	"time"
)

// RetryConfig holds retry strategy parameters
type RetryConfig struct {
	MaxRetries        int           // Maximum number of retry attempts
	InitialBackoff    time.Duration // Initial backoff duration
	MaxBackoff        time.Duration // Maximum backoff duration
	BackoffMultiplier float64       // Multiplier for exponential backoff
}

// DefaultRetryConfig provides sensible defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        5 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// Retry executes a function with exponential backoff and jitter
func Retry(ctx context.Context, config RetryConfig, fn func(ctx context.Context) error) error {
	var lastErr error
	backoff := config.InitialBackoff

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try the operation
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't backoff on last attempt
		if attempt == config.MaxRetries {
			break
		}

		// Calculate backoff with jitter: backoff * (0.5 + 0.5*random)
		jitter := backoff * time.Duration(0.5+0.5*rand.Float64())
		if jitter > config.MaxBackoff {
			jitter = config.MaxBackoff
		}

		// Wait with context awareness
		select {
		case <-time.After(jitter):
		case <-ctx.Done():
			return ctx.Err()
		}

		// Exponential backoff for next attempt
		backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
		}
	}

	return lastErr
}
