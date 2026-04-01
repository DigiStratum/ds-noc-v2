// Package middleware tests for correlation, logging, and recovery middleware.
// Tests NFR-MON-001, NFR-MON-004, NFR-AVAIL-001
package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Tests NFR-MON-004: Correlation ID is added to each request
func TestCorrelationIDMiddleware_GeneratesNewID(t *testing.T) {
	// Arrange
	handler := CorrelationIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := GetCorrelationID(r.Context())
		if correlationID == "" {
			t.Error("expected correlation ID in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	responseID := rec.Header().Get("X-Correlation-ID")
	if responseID == "" {
		t.Error("expected X-Correlation-ID response header")
	}

	// Should be a valid UUID format (36 chars with dashes)
	if len(responseID) != 36 {
		t.Errorf("expected UUID format, got %q", responseID)
	}
}

// Tests NFR-MON-004: Existing correlation ID is preserved from upstream
func TestCorrelationIDMiddleware_PreservesExistingID(t *testing.T) {
	existingID := "upstream-correlation-id-12345"

	handler := CorrelationIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := GetCorrelationID(r.Context())
		if correlationID != existingID {
			t.Errorf("expected %q, got %q", existingID, correlationID)
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", existingID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Correlation-ID") != existingID {
		t.Error("expected upstream correlation ID to be preserved")
	}
}

// Tests NFR-MON-004: AWS request ID is used when X-Correlation-ID is absent
func TestCorrelationIDMiddleware_UsesAWSRequestID(t *testing.T) {
	awsRequestID := "aws-request-id-xyz"

	handler := CorrelationIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := GetCorrelationID(r.Context())
		if correlationID != awsRequestID {
			t.Errorf("expected AWS request ID %q, got %q", awsRequestID, correlationID)
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Amzn-Request-Id", awsRequestID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
}

func TestGetCorrelationID_EmptyContext(t *testing.T) {
	ctx := context.Background()
	id := GetCorrelationID(ctx)
	if id != "" {
		t.Errorf("expected empty string for context without correlation ID, got %q", id)
	}
}

func TestLoggerWithCorrelation_AddsCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{})
	slog.SetDefault(slog.New(handler))

	// Create context with correlation ID
	ctx := context.WithValue(context.Background(), correlationIDKey, "test-correlation-123")

	logger := LoggerWithCorrelation(ctx)
	logger.Info("test message")

	logOutput := buf.String()
	if !strings.Contains(logOutput, "test-correlation-123") {
		t.Errorf("expected log to contain correlation ID, got: %s", logOutput)
	}
}

// Tests NFR-MON-001: Request completion is logged with duration and status
func TestLoggingMiddleware_LogsRequestCompletion(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{})
	slog.SetDefault(slog.New(handler))

	middleware := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	}))

	req := httptest.NewRequest("GET", "/test-path", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	logOutput := buf.String()

	// Should log "request completed" with required fields
	if !strings.Contains(logOutput, "request completed") {
		t.Errorf("expected 'request completed' in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "/test-path") {
		t.Errorf("expected path in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "duration_ms") {
		t.Errorf("expected duration_ms in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, `"status":200`) {
		t.Errorf("expected status 200 in log, got: %s", logOutput)
	}
}

// Tests NFR-MON-001: Error level logging for 5xx responses
func TestLoggingMiddleware_ErrorLevelFor500(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	middleware := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, `"level":"ERROR"`) {
		t.Errorf("expected ERROR level for 500 response, got: %s", logOutput)
	}
}

// Tests NFR-MON-001: Warn level logging for 4xx responses
func TestLoggingMiddleware_WarnLevelFor400(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))

	middleware := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, `"level":"WARN"`) {
		t.Errorf("expected WARN level for 404 response, got: %s", logOutput)
	}
}

func TestLoggingMiddleware_CapturesResponseSize(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{})
	slog.SetDefault(slog.New(handler))

	responseBody := "hello world response"

	middleware := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(responseBody))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	logOutput := buf.String()
	// Should include size_bytes field
	if !strings.Contains(logOutput, `"size_bytes":`) {
		t.Errorf("expected size_bytes in log, got: %s", logOutput)
	}
}

// Tests NFR-AVAIL-001: Panic recovery returns 500 error
func TestRecoveryMiddleware_RecoversPanic(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{})
	slog.SetDefault(slog.New(handler))

	middleware := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic message")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Should not panic
	middleware.ServeHTTP(rec, req)

	// Should return 500
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	// Should return JSON error response
	body := rec.Body.String()
	if !strings.Contains(body, "INTERNAL_ERROR") {
		t.Errorf("expected INTERNAL_ERROR in response, got: %s", body)
	}

	// Should log the panic
	logOutput := buf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Errorf("expected 'panic recovered' in log, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "test panic message") {
		t.Errorf("expected panic message in log, got: %s", logOutput)
	}
}

func TestRecoveryMiddleware_PassesThroughNormalRequests(t *testing.T) {
	middleware := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("normal response"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != "normal response" {
		t.Errorf("expected 'normal response', got %q", rec.Body.String())
	}
}

func TestRecoveryMiddleware_IncludesCorrelationID(t *testing.T) {
	correlationID := "test-correlation-456"

	// Chain correlation middleware before recovery to test integration
	handler := CorrelationIDMiddleware(RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Correlation-ID", correlationID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Response should include correlation ID
	if rec.Header().Get("X-Correlation-ID") != correlationID {
		t.Errorf("expected correlation ID in response header")
	}

	// Error response should include request_id
	body := rec.Body.String()
	if !strings.Contains(body, correlationID) {
		t.Errorf("expected correlation ID in error response, got: %s", body)
	}
}

// Test responseWriter wrapper
func TestResponseWriter_CapturesStatus(t *testing.T) {
	inner := httptest.NewRecorder()
	wrapped := wrapResponseWriter(inner)

	wrapped.WriteHeader(http.StatusCreated)

	if wrapped.status != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, wrapped.status)
	}
}

func TestResponseWriter_DefaultStatusIsOK(t *testing.T) {
	inner := httptest.NewRecorder()
	wrapped := wrapResponseWriter(inner)

	// Write without explicit WriteHeader should default to 200
	if wrapped.status != http.StatusOK {
		t.Errorf("expected default status 200, got %d", wrapped.status)
	}
}

func TestResponseWriter_PreventsDuplicateWriteHeader(t *testing.T) {
	inner := httptest.NewRecorder()
	wrapped := wrapResponseWriter(inner)

	wrapped.WriteHeader(http.StatusCreated)
	wrapped.WriteHeader(http.StatusBadRequest) // Should be ignored

	if wrapped.status != http.StatusCreated {
		t.Errorf("expected first status %d to be preserved, got %d", http.StatusCreated, wrapped.status)
	}
}

func TestResponseWriter_TracksWriteSize(t *testing.T) {
	inner := httptest.NewRecorder()
	wrapped := wrapResponseWriter(inner)

	data1 := []byte("hello")
	data2 := []byte(" world")

	_, _ = wrapped.Write(data1)
	_, _ = wrapped.Write(data2)

	if wrapped.size != len(data1)+len(data2) {
		t.Errorf("expected size %d, got %d", len(data1)+len(data2), wrapped.size)
	}
}
