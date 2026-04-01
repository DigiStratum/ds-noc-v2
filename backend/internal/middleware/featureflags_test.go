package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFeatureFlags_InjectsContext(t *testing.T) {
	var capturedCtx context.Context
	
	handler := FeatureFlags()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that evaluation context was injected
	evalCtx := GetEvaluationContext(capturedCtx)
	if evalCtx == nil {
		t.Error("expected evaluation context to be injected")
	}
}

func TestIsEnabled_NoMiddleware(t *testing.T) {
	// Without middleware, IsEnabled should return false
	ctx := context.Background()
	
	if IsEnabled(ctx, "any-flag") {
		t.Error("expected IsEnabled to return false without middleware")
	}
}

func TestIsEnabledForUser_EmptyContext(t *testing.T) {
	ctx := context.Background()
	
	// This calls the evaluator directly without middleware
	// With an empty store, all flags should be disabled
	result := IsEnabledForUser(ctx, "test-flag", "user-1", "session-1", "tenant-1")
	
	// Note: This may panic if the store isn't properly initialized
	// In a real test, you'd mock the store
	if result {
		t.Error("expected IsEnabledForUser to return false for non-existent flag")
	}
}

func TestGetEvaluationContext_NotSet(t *testing.T) {
	ctx := context.Background()
	
	evalCtx := GetEvaluationContext(ctx)
	if evalCtx != nil {
		t.Error("expected nil evaluation context when not set")
	}
}
