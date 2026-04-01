package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/session"
)

func TestListSessions_NoSession(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/session", nil)
	w := httptest.NewRecorder()

	ListSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response SessionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.IsAuthenticated {
		t.Error("expected is_authenticated=false for no session")
	}
	if !response.IsGuest {
		t.Error("expected is_guest=true for no session")
	}
}

func TestListSessions_GuestSession(t *testing.T) {
	sess := &session.Session{
		ID:      "test-session-id-12345",
		IsGuest: true,
	}

	req := httptest.NewRequest("GET", "/api/session", nil)
	ctx := session.SetSession(req.Context(), sess)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ListSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response SessionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.IsAuthenticated {
		t.Error("expected is_authenticated=false for guest session")
	}
	if !response.IsGuest {
		t.Error("expected is_guest=true for guest session")
	}
	if response.SessionID != "test-ses..." {
		t.Errorf("expected truncated session ID 'test-ses...', got %q", response.SessionID)
	}
}

func TestListSessions_AuthenticatedSession(t *testing.T) {
	sess := &session.Session{
		ID:       "test-session-id-12345",
		UserID:   "user-123",
		IsGuest:  false,
		TenantID: "org:acme",
	}

	req := httptest.NewRequest("GET", "/api/session", nil)
	ctx := session.SetSession(req.Context(), sess)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	ListSessions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response SessionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !response.IsAuthenticated {
		t.Error("expected is_authenticated=true for authenticated session")
	}
	if response.IsGuest {
		t.Error("expected is_guest=false for authenticated session")
	}
	if response.TenantID != "org:acme" {
		t.Errorf("expected tenant_id='org:acme', got %q", response.TenantID)
	}
}

func TestTruncateSessionID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"abcdefgh12345", "abcdefgh..."},
		{"short", "short"},
		{"12345678", "12345678"},
		{"123456789", "12345678..."},
		{"", ""},
	}

	for _, tc := range tests {
		got := truncateSessionID(tc.input)
		if got != tc.expected {
			t.Errorf("truncateSessionID(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// Helper to add session to context for testing
func withSession(ctx context.Context, sess *session.Session) context.Context {
	return session.SetSession(ctx, sess)
}
