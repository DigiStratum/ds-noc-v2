package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	size        int
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.size += len(b)
	return rw.ResponseWriter.Write(b)
}

// LoggingMiddleware logs request completion with duration and status [NFR-MON-001]
// Produces CloudWatch-compatible structured JSON logs
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := wrapResponseWriter(w)

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		correlationID := GetCorrelationID(r.Context())

		// Use appropriate log level based on status code
		logLevel := slog.LevelInfo
		if wrapped.status >= 500 {
			logLevel = slog.LevelError
		} else if wrapped.status >= 400 {
			logLevel = slog.LevelWarn
		}

		// Structured log for CloudWatch Logs Insights [NFR-MON-001]
		slog.Log(r.Context(), logLevel, "request completed",
			"correlation_id", correlationID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration_ms", duration.Milliseconds(),
			"size_bytes", wrapped.size,
		)
	})
}
