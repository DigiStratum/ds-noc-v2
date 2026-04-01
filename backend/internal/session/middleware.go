// Package session middleware provides guest-session-first authentication.
// See session.go for the session model and store.
package session

import (
	"log/slog"
	"net/http"
)

// Middleware ensures every request has a session (anonymous or authenticated).
// This is the core of the guest-session-first pattern:
// - Check for existing session from cookie
// - If no session exists, create an anonymous one
// - Auth endpoints handle upgrading anonymous sessions to authenticated
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		store := GetStore()
		tenantID := r.Header.Get("X-Tenant-ID")

		// Try to get session ID from request
		sessionID := GetSessionIDFromRequest(r)

		// Try local session store
		session := store.Get(sessionID)

		// Create anonymous session if none exists or expired
		if session == nil {
			session = store.Create(tenantID)

			// Set cookie for new session
			SetSessionCookie(w, r, session)

			slog.Info("created anonymous session",
				"session_id", session.ID[:8]+"...",
				"tenant_id", tenantID,
			)
		}

		// Add session to context
		ctx := SetSession(r.Context(), session)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth is middleware that requires an authenticated session.
// Use this for routes that need a logged-in user.
// Anonymous sessions get a 401 response (for API) or redirect (for browser).
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(r.Context())

		if session == nil || !session.IsAuthenticated() {
			// Check if this is an API request or browser request
			if isAPIRequest(r) {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"Authentication required"}}`, http.StatusUnauthorized)
			} else {
				// Redirect to login page with return URL
				loginURL := "/api/auth/login?redirect=" + r.URL.Path
				http.Redirect(w, r, loginURL, http.StatusFound)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isAPIRequest checks if the request is for the API (vs browser navigation)
func isAPIRequest(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	contentType := r.Header.Get("Content-Type")

	// API requests typically want JSON
	if accept == "application/json" || contentType == "application/json" {
		return true
	}

	// API requests usually come from fetch/XHR
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		return true
	}

	return false
}
