package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/auth"
)

func TestListMes_NoUser(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/me", nil)
	w := httptest.NewRecorder()

	ListMes(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var response ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Error.Code != "UNAUTHORIZED" {
		t.Errorf("expected error code 'UNAUTHORIZED', got %q", response.Error.Code)
	}
}

func TestListMes_WithUser(t *testing.T) {
	user := &auth.User{
		ID:    "user-123",
		Email: "test@example.com",
		Name:  "Test User",
	}

	req := httptest.NewRequest("GET", "/api/me", nil)
	ctx := auth.SetUser(req.Context(), user)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ListMes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response auth.User
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.ID != "user-123" {
		t.Errorf("expected user ID 'user-123', got %q", response.ID)
	}
	if response.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", response.Email)
	}
}

// withUser adds a user to the context for testing.
func withUser(ctx context.Context, user *auth.User) context.Context {
	return auth.SetUser(ctx, user)
}
