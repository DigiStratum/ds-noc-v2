// Updated in template v0.5.0 - health check improvements
// Package health provides the HTTP handler for health checks.
package health

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/auth"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/session"
)

// Handler handles health check requests.
// - GET /health or GET /health?depth=shallow: Quick check, no auth required
// - GET /health?depth=deep: Full check with dependencies, requires M2M token or superadmin
func Handler(w http.ResponseWriter, r *http.Request) {
	depth := Depth(r.URL.Query().Get("depth"))
	
	// Default to shallow
	if depth == "" || depth == DepthShallow {
		handleShallow(w, r)
		return
	}
	
	if depth == DepthDeep {
		handleDeep(w, r)
		return
	}
	
	// Invalid depth parameter
	WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
		"error": map[string]string{
			"code":    "INVALID_DEPTH",
			"message": "depth must be 'shallow' or 'deep'",
		},
	})
}

// handleShallow performs a quick health check without authentication.
func handleShallow(w http.ResponseWriter, r *http.Request) {
	resp := ShallowCheck()
	
	status := http.StatusOK
	if resp.Status == StatusDown {
		status = http.StatusServiceUnavailable
	}
	
	WriteJSON(w, status, resp)
}

// handleDeep performs a comprehensive health check with authentication.
func handleDeep(w http.ResponseWriter, r *http.Request) {
	logger := middleware.LoggerWithCorrelation(r.Context())
	
	// Check authentication for deep health checks
	if !isAuthorizedForDeepCheck(r) {
		logger.Warn("unauthorized deep health check attempt",
			"remote_addr", r.RemoteAddr,
		)
		WriteJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"error": map[string]string{
				"code":    "UNAUTHORIZED",
				"message": "Deep health checks require M2M token or superadmin session",
			},
		})
		return
	}
	
	logger.Info("performing deep health check")
	
	resp := DeepCheck(r.Context())
	
	status := http.StatusOK
	switch resp.Status {
	case StatusDegraded:
		status = http.StatusOK // Still 200, but status field indicates degraded
	case StatusDown:
		status = http.StatusServiceUnavailable
	}
	
	logger.Info("deep health check completed",
		"status", resp.Status,
		"dependencies_count", len(resp.Dependencies),
	)
	
	WriteJSON(w, status, resp)
}

// isAuthorizedForDeepCheck checks if the request is authorized for deep health checks.
// Accepts:
// - M2M token in Authorization header (Bearer token with m2m scope)
// - Superadmin session (user with superadmin role)
func isAuthorizedForDeepCheck(r *http.Request) bool {
	// Check for M2M token first
	if isValidM2MToken(r) {
		return true
	}
	
	// Check for superadmin session
	if isSuperadminSession(r) {
		return true
	}
	
	return false
}

// isValidM2MToken checks if the request has a valid machine-to-machine token.
// M2M tokens are typically used by:
// - Monitoring systems
// - Other internal services
// - CI/CD pipelines
func isValidM2MToken(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}
	
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return false
	}
	
	// Check for M2M token marker
	// In production, this would validate against DSAccount's token validation endpoint
	// For the boilerplate, we accept tokens with a specific prefix or check a claim
	
	// Option 1: Check against configured M2M token (simple approach)
	configuredToken := os.Getenv("HEALTH_M2M_TOKEN")
	if configuredToken != "" && token == configuredToken {
		return true
	}
	
	// Option 2: Validate JWT and check for m2m scope
	// This would be the production implementation:
	// claims, err := validateJWT(token)
	// if err != nil {
	//     return false
	// }
	// return claims.HasScope("health:deep")
	
	return false
}

// isSuperadminSession checks if the current session has superadmin privileges.
func isSuperadminSession(r *http.Request) bool {
	sess := session.GetSession(r.Context())
	if sess == nil || !sess.IsAuthenticated() {
		return false
	}
	
	user := auth.GetUser(r.Context())
	if user == nil {
		return false
	}
	
	// Check for superadmin role
	// In production, this would check a roles/permissions claim
	// For the boilerplate, we check for a specific user ID or email pattern
	
	// Check against configured superadmin IDs
	superadminIDsStr := os.Getenv("SUPERADMIN_USER_IDS")
	if superadminIDsStr != "" {
		superadminIDs := strings.Split(superadminIDsStr, ",")
		for _, id := range superadminIDs {
			if user.ID == strings.TrimSpace(id) {
				return true
			}
		}
	}
	
	// Check for superadmin email pattern (e.g., *+superadmin@digistratum.com)
	if strings.Contains(user.Email, "+superadmin@") {
		return true
	}
	
	return false
}

// LoggerStub provides a fallback logger when middleware isn't available.
// This is used for testing or when health checks are called outside the middleware chain.
func init() {
	// Ensure slog has a default logger
	_ = slog.Default()
}
