package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const correlationIDKey contextKey = "correlation_id"

// CorrelationIDMiddleware adds a correlation ID to each request [NFR-MON-004]
// The ID is either extracted from X-Correlation-ID header (for distributed tracing)
// or generated fresh for new requests.
func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing correlation ID (from upstream service or API Gateway)
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			// Also check AWS request ID from API Gateway
			correlationID = r.Header.Get("X-Amzn-Request-Id")
		}
		if correlationID == "" {
			// Generate new UUID
			correlationID = uuid.New().String()
		}

		// Add to context
		ctx := context.WithValue(r.Context(), correlationIDKey, correlationID)

		// Add to response headers for client-side correlation
		w.Header().Set("X-Correlation-ID", correlationID)

		// Log request start with correlation ID
		slog.Info("request started",
			"correlation_id", correlationID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCorrelationID retrieves the correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// LoggerWithCorrelation returns a slog.Logger with the correlation ID pre-attached
// Use this in handlers for consistent structured logging
func LoggerWithCorrelation(ctx context.Context) *slog.Logger {
	correlationID := GetCorrelationID(ctx)
	return slog.Default().With("correlation_id", correlationID)
}
