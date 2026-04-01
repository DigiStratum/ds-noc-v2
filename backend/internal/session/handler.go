// Package session provides session HTTP handler
package session

import (
	"encoding/json"
	"net/http"
)

// SessionResponse represents the session endpoint response
type SessionResponse struct {
	Session       *Session `json:"session,omitempty"`
	Authenticated bool     `json:"authenticated"`
}

// Handler returns the current session state
// GET /api/session - returns session info including auth status
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sess := GetSession(r.Context())

	response := SessionResponse{
		Session:       sess,
		Authenticated: sess != nil && sess.IsAuthenticated(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
