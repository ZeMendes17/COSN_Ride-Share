package request

import (
	"context"
	"errors"
	"github.com/go-kit/kit/endpoint"
)

var (
	ErrUnauthorized = errors.New("authentication required")
	ErrForbidden    = errors.New("you are not allowed to access this resource")
)

// AuthMiddleware ensures that a user ID is present in the context.
func AuthMiddleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			userID, ok := ctx.Value(ContextKeyUserID).(string)
			if !ok || userID == "" {
				return nil, ErrUnauthorized
			}
			// Proceed to the next step
			return next(ctx, request)
		}
	}
}

// Context keys
type contextKey string

const ContextKeyUserID contextKey = "userID"