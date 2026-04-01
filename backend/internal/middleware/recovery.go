package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// RecoveryMiddleware catches panics and returns a 500 error [NFR-AVAIL-001]
// Logs the panic with stack trace for debugging
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Get correlation ID from context if available
				correlationID := GetCorrelationID(r.Context())
				
				// Log the panic with full context
				slog.Error("panic recovered",
					"error", fmt.Sprintf("%v", err),
					"correlation_id", correlationID,
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)

				// Return standardized error response
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Correlation-ID", correlationID)
				w.WriteHeader(http.StatusInternalServerError)
				
				// Use standard error format [NFR-SEC-004]
				_, _ = fmt.Fprintf(w, `{"error":{"code":"INTERNAL_ERROR","message":"An unexpected error occurred","request_id":"%s"}}`, correlationID)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
