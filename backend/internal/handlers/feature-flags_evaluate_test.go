package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEvaluateFeatureFlags(t *testing.T) {
	// Note: This test will fail in isolation since it requires DynamoDB
	// In real tests, we would mock the store
	t.Skip("Requires DynamoDB connection - integration test")
	
	req := httptest.NewRequest("GET", "/api/feature-flags/evaluate", nil)
	w := httptest.NewRecorder()

	EvaluateFeatureFlags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response is HAL-compliant
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/hal+json" {
		t.Errorf("expected Content-Type application/hal+json, got %s", contentType)
	}
}

func TestBuildEvaluationContext(t *testing.T) {
	t.Run("extracts session from cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feature-flags/evaluate", nil)
		req.AddCookie(&http.Cookie{Name: "ds_session", Value: "test-session-id"})

		ctx := buildEvaluationContext(req)

		if ctx.SessionID != "test-session-id" {
			t.Errorf("expected SessionID %q, got %q", "test-session-id", ctx.SessionID)
		}
	})

	t.Run("extracts tenant from header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feature-flags/evaluate", nil)
		req.Header.Set("X-Tenant-ID", "test-tenant")

		ctx := buildEvaluationContext(req)

		if ctx.TenantID != "test-tenant" {
			t.Errorf("expected TenantID %q, got %q", "test-tenant", ctx.TenantID)
		}
	})

	t.Run("handles missing values gracefully", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/feature-flags/evaluate", nil)

		ctx := buildEvaluationContext(req)

		if ctx.UserID != "" || ctx.SessionID != "" || ctx.TenantID != "" {
			t.Error("expected empty evaluation context for request without auth data")
		}
	})
}
