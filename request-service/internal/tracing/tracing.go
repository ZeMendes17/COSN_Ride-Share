package tracing

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

const (
	TraceIDHeader = "X-Trace-ID"
	SpanIDHeader  = "X-Span-ID"
)

// TraceContext holds trace information
type TraceContext struct {
	TraceID string
	SpanID  string
}

// ContextKeyTraceID is the key for storing trace context in context.Context
type ContextKeyTraceID struct{}

// GenerateTraceID creates a new trace ID
func GenerateTraceID() string {
	return uuid.New().String()
}

// GenerateSpanID creates a new span ID
func GenerateSpanID() string {
	return uuid.New().String()
}

// ExtractTraceContext extracts or creates trace context from HTTP request
func ExtractTraceContext(r *http.Request) TraceContext {
	traceID := r.Header.Get(TraceIDHeader)
	if traceID == "" {
		traceID = GenerateTraceID()
	}

	spanID := r.Header.Get(SpanIDHeader)
	if spanID == "" {
		spanID = GenerateSpanID()
	}

	return TraceContext{
		TraceID: traceID,
		SpanID:  spanID,
	}
}

// InjectTraceContext adds trace headers to an HTTP request
func InjectTraceContext(req *http.Request, tc TraceContext) {
	req.Header.Set(TraceIDHeader, tc.TraceID)
	req.Header.Set(SpanIDHeader, tc.SpanID)
}

// TraceContextToContext stores trace context in context.Context
func TraceContextToContext(ctx context.Context, tc TraceContext) context.Context {
	return context.WithValue(ctx, ContextKeyTraceID{}, tc)
}

// TraceContextFromContext retrieves trace context from context.Context
func TraceContextFromContext(ctx context.Context) (TraceContext, bool) {
	tc, ok := ctx.Value(ContextKeyTraceID{}).(TraceContext)
	return tc, ok
}
