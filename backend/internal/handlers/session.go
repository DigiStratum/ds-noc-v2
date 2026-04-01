package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/auth"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/session"
)

// SessionResponse represents the session state for the frontend
type SessionResponse struct {
	SessionID       string     `json:"session_id,omitempty"`
	IsAuthenticated bool       `json:"is_authenticated"`
	IsGuest         bool       `json:"is_guest"`
	TenantID        string     `json:"tenant_id,omitempty"`
	User            *auth.User `json:"user,omitempty"`
}

// ListSessions handles GET /api/session
// Returns the current session state (works for both guest and authenticated users).
// This is the primary endpoint for the frontend to check session status.
func ListSessions(w http.ResponseWriter, r *http.Request) {
	// Set txnlog data type for transaction logging
	middleware.SetTxnLogDataType(r.Context(), "Session")

	sess := session.GetSession(r.Context())

	// No session - return guest state
	if sess == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SessionResponse{
			IsAuthenticated: false,
			IsGuest:         true,
		})
		return
	}

	// Build response based on session state
	response := SessionResponse{
		SessionID:       truncateSessionID(sess.ID),
		IsAuthenticated: sess.IsAuthenticated(),
		IsGuest:         sess.IsGuest,
		TenantID:        sess.TenantID,
	}

	// Include user if authenticated
	if sess.IsAuthenticated() {
		response.User = auth.GetUser(r.Context())
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// truncateSessionID returns a truncated session ID for display (security)
func truncateSessionID(id string) string {
	if len(id) > 8 {
		return id[:8] + "..."
	}
	return id
}
