package auth

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/session"
	"github.com/DigiStratum/ds-noc-v2/backend/pkg/tenant"
)

type contextKey string

const (
	userContextKey contextKey = "user"
)

// User represents an authenticated user [FR-AUTH-003]
type User struct {
	ID      string       `json:"id"`
	Email   string       `json:"email"`
	Name    string       `json:"display_name"` // DSAccount uses display_name
	Tenants []TenantInfo `json:"tenants"`
}

// TenantInfo represents a user's membership in a tenant/org
type TenantInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
	Type string `json:"type"` // "user" or "org"
}

// Middleware validates authentication and extracts user/tenant context [FR-AUTH-001]
// This works with the session middleware to support both guest and authenticated sessions.
//
// Guest session pattern:
// - Session middleware runs first, ensuring every request has a session
// - This middleware enriches the context with user data if session is authenticated
// - Unauthenticated sessions are allowed to pass through (guest mode)
//
// API key authentication (M2M/CLI):
// - Bearer token matching COMPONENTS_API_KEY env var creates system user
// - Used by ds-components CLI for component registry operations
//
// Tenant Context:
// - Tenant is extracted from session (authenticated users) or X-Tenant-ID header
// - Uses pkg/tenant for type-safe tenant handling
// - All tenant-scoped data access MUST use GetTenant(ctx) for isolation
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.GetSession(r.Context())

		var user *User
		var t tenant.Tenant

		// Check for API key authentication (M2M/CLI access)
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			apiKey := os.Getenv("COMPONENTS_API_KEY")
			if apiKey != "" && token == apiKey {
				// API key auth - create synthetic user for M2M access
				user = &User{
					ID:    "system:cli",
					Email: "cli@digistratum.com",
					Name:  "DS Components CLI",
				}
				// System user gets a system tenant for audit purposes
				t = tenant.Tenant{Type: tenant.TenantTypeUser, ID: "system"}
				slog.Debug("authenticated via API key", "user_id", user.ID)
			}
		}

		// If we have an authenticated session, load the user
		if user == nil && sess != nil && sess.IsAuthenticated() {
			var err error
			user, err = loadUser(sess.UserID)
			if err != nil {
				slog.Warn("failed to load user for session", "user_id", sess.UserID, "error", err)
				// Don't fail the request - just proceed as guest
			}

			// Extract tenant from session
			if sess.TenantID != "" {
				var err error
				t, err = tenant.Parse(sess.TenantID)
				if err != nil {
					slog.Warn("invalid tenant in session, falling back to header",
						"session_tenant", sess.TenantID,
						"error", err,
					)
				}
			}
		}

		// Fall back to header if no tenant from session [FR-TENANT-004]
		if t.IsZero() {
			headerVal := r.Header.Get("X-Tenant-ID")
			if headerVal != "" {
				var err error
				t, err = tenant.Parse(headerVal)
				if err != nil {
					slog.Warn("invalid tenant header", "header", headerVal, "error", err)
					// Continue without tenant - RequireTenantMiddleware will reject if needed
				}
			}
		}

		// Add user and tenant to context
		ctx := r.Context()
		if user != nil {
			ctx = context.WithValue(ctx, userContextKey, user)
		}
		if !t.IsZero() {
			ctx = tenant.SetTenant(ctx, t)

			// Log for audit trail
			slog.Debug("tenant context set",
				"tenant", t.String(),
				"path", r.URL.Path,
				"user_id", getUserID(user),
			)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getUserID safely extracts user ID for logging
func getUserID(u *User) string {
	if u == nil {
		return ""
	}
	return u.ID
}

// RequireAuthMiddleware requires an authenticated user [FR-AUTH-002]
// Use this for routes that need a logged-in user (not just a session).
func RequireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r.Context())

		if user == nil {
			// Redirect to SSO login [FR-AUTH-002]
			// SECURITY: Only app_id is passed. redirect_uri comes from DSAccount app registration.
			ssoURL := os.Getenv("DSACCOUNT_SSO_URL")
			if ssoURL == "" {
				ssoURL = "https://account.digistratum.com"
			}
			redirectURL := ssoURL + "/api/sso/authorize?app_id=" + os.Getenv("DSACCOUNT_APP_ID")
			http.Redirect(w, r, redirectURL, http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireTenantMiddleware requires tenant context on the request.
// Use this for routes that must be tenant-scoped.
// Rejects with 401 if no tenant is set.
func RequireTenantMiddleware(next http.Handler) http.Handler {
	return tenant.RequireTenantMiddleware(next)
}

// GetUser extracts user from context
func GetUser(ctx context.Context) *User {
	user, _ := ctx.Value(userContextKey).(*User)
	return user
}

// SetUser sets the user in context (primarily for testing)
func SetUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetTenant extracts tenant from context using the tenant package.
// Returns zero Tenant if not set.
//
// For handlers that require tenant, use RequireTenant instead.
func GetTenant(ctx context.Context) tenant.Tenant {
	return tenant.GetTenant(ctx)
}

// RequireTenant extracts tenant from context, returning error if not set.
// Use this in handlers/services that require tenant isolation.
func RequireTenant(ctx context.Context) (tenant.Tenant, error) {
	return tenant.RequireTenant(ctx)
}

// GetTenantID returns the canonical tenant string (e.g., "user:123" or "org:abc").
// Returns empty string if no tenant in context.
//
// Deprecated: Use GetTenant(ctx).String() instead for the canonical form,
// or GetTenant(ctx) to access the full Tenant struct.
func GetTenantID(ctx context.Context) string {
	t := tenant.GetTenant(ctx)
	if t.IsZero() {
		return ""
	}
	return t.String()
}

// loadUser loads user data from DSAccount or cache
func loadUser(userID string) (*User, error) {
	// TODO: Implement actual DSAccount user lookup
	// For the boilerplate, return mock user
	return &User{
		ID:    userID,
		Email: "demo@digistratum.com",
		Name:  "Demo User",
		Tenants: []TenantInfo{
			{ID: "tenant-1", Name: "Demo Tenant", Role: "member", Type: "org"},
		},
	}, nil
}
