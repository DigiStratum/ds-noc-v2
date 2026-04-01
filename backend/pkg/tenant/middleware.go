// Package tenant provides multi-tenant isolation primitives.
//
// This file contains HTTP middleware for tenant extraction and validation.
package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// Middleware options for tenant extraction
type MiddlewareConfig struct {
	// AllowAnonymous allows requests without tenant context to pass through.
	// Default: false (strict mode - reject if no tenant)
	AllowAnonymous bool

	// HeaderName is the HTTP header to check for tenant ID.
	// Default: "X-Tenant-ID"
	HeaderName string

	// TenantExtractor is called to extract tenant from custom sources.
	// If nil, only header and session are checked.
	TenantExtractor func(r *http.Request) (Tenant, bool)

	// OnError is called when tenant validation fails.
	// If nil, returns 401 Unauthorized JSON response.
	OnError func(w http.ResponseWriter, r *http.Request, err error)
}

// DefaultConfig returns the default middleware configuration.
func DefaultConfig() MiddlewareConfig {
	return MiddlewareConfig{
		AllowAnonymous: false,
		HeaderName:     "X-Tenant-ID",
	}
}

// Middleware creates HTTP middleware that extracts and validates tenant context.
//
// Tenant resolution order:
//  1. TenantExtractor (if configured) - for session/JWT-based resolution
//  2. X-Tenant-ID header (or configured HeaderName)
//
// If no tenant is found and AllowAnonymous is false, returns 401.
//
// Usage:
//
//	handler := tenant.Middleware(tenant.DefaultConfig())(yourHandler)
func Middleware(config MiddlewareConfig) func(http.Handler) http.Handler {
	if config.HeaderName == "" {
		config.HeaderName = "X-Tenant-ID"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var t Tenant
			var found bool

			// 1. Try custom extractor first (e.g., from session/JWT)
			if config.TenantExtractor != nil {
				t, found = config.TenantExtractor(r)
			}

			// 2. Fall back to header
			if !found {
				headerVal := r.Header.Get(config.HeaderName)
				if headerVal != "" {
					var err error
					t, err = Parse(headerVal)
					if err != nil {
						handleError(w, r, config, fmt.Errorf("invalid tenant header: %w", err))
						return
					}
					found = true
				}
			}

			// 3. Check if tenant is required
			if !found {
				if config.AllowAnonymous {
					// Allow through without tenant
					slog.Debug("request proceeding without tenant (anonymous allowed)")
					next.ServeHTTP(w, r)
					return
				}

				handleError(w, r, config, ErrNoTenantInContext)
				return
			}

			// Validate tenant is complete
			if t.IsZero() {
				handleError(w, r, config, fmt.Errorf("tenant is incomplete: %+v", t))
				return
			}

			// Add tenant to context
			ctx := SetTenant(r.Context(), t)

			// Log tenant for audit trail
			slog.Debug("tenant context set",
				"tenant_type", t.Type,
				"tenant_id", t.ID,
				"path", r.URL.Path,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// handleError responds with an appropriate error based on config
func handleError(w http.ResponseWriter, r *http.Request, config MiddlewareConfig, err error) {
	slog.Warn("tenant validation failed",
		"error", err,
		"path", r.URL.Path,
		"method", r.Method,
	)

	if config.OnError != nil {
		config.OnError(w, r, err)
		return
	}

	// Default error response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    "TENANT_REQUIRED",
			"message": "Tenant context is required for this request",
		},
	})
}

// RequireTenantMiddleware is a simpler middleware that just requires tenant in context.
// Use this after auth middleware has already set the tenant.
//
// Unlike Middleware(), this doesn't extract tenant - it just validates it exists.
func RequireTenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := GetTenant(r.Context())
		if t.IsZero() {
			slog.Warn("request missing required tenant context",
				"path", r.URL.Path,
				"method", r.Method,
			)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    "TENANT_REQUIRED",
					"message": "Tenant context is required for this request",
				},
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WithSessionTenantExtractor creates a config that extracts tenant from session.
// This is the common pattern where auth middleware sets session, then tenant
// middleware reads tenant from that session.
//
// sessionGetter should return (tenantString, userID, ok) from the session in context.
func WithSessionTenantExtractor(sessionGetter func(ctx context.Context) (tenantStr, userID string, ok bool)) MiddlewareConfig {
	config := DefaultConfig()
	config.TenantExtractor = func(r *http.Request) (Tenant, bool) {
		tenantStr, _, ok := sessionGetter(r.Context())
		if !ok || tenantStr == "" {
			return Tenant{}, false
		}

		t, err := Parse(tenantStr)
		if err != nil {
			slog.Warn("invalid tenant in session", "tenant", tenantStr, "error", err)
			return Tenant{}, false
		}

		return t, true
	}
	return config
}

// TenantScopeHandler wraps a handler to automatically validate tenant scope.
// Use for handlers that MUST have tenant context.
//
// Example:
//
//	mux.Handle("GET /api/issues", tenant.TenantScopeHandler(issueHandler))
func TenantScopeHandler(handler http.Handler) http.Handler {
	return RequireTenantMiddleware(handler)
}

// PublicRoutes is middleware that skips tenant validation for specified paths.
// Use this to allow health checks and public endpoints without tenant.
//
// Example:
//
//	handler := tenant.PublicRoutes(
//	    []string{"/api/health", "/api/discovery"},
//	    tenant.Middleware(config),
//	)(appHandler)
func PublicRoutes(publicPaths []string, tenantMiddleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	// Build set for O(1) lookup
	publicSet := make(map[string]bool)
	for _, p := range publicPaths {
		publicSet[p] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is public
			path := r.URL.Path
			if publicSet[path] {
				next.ServeHTTP(w, r)
				return
			}

			// Check prefixes (for patterns like /api/public/*)
			for p := range publicSet {
				if strings.HasSuffix(p, "*") {
					prefix := strings.TrimSuffix(p, "*")
					if strings.HasPrefix(path, prefix) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			// Apply tenant middleware
			tenantMiddleware(next).ServeHTTP(w, r)
		})
	}
}

// SkipTenantForCLI allows bypassing tenant validation for CLI/M2M access
// when using a valid API key. Use sparingly and audit all CLI access.
func SkipTenantForCLI(apiKeyEnvVar string, config MiddlewareConfig) MiddlewareConfig {
	originalExtractor := config.TenantExtractor
	config.TenantExtractor = func(r *http.Request) (Tenant, bool) {
		// Check for API key auth first
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			apiKey := os.Getenv(apiKeyEnvVar)
			if apiKey != "" && token == apiKey {
				// CLI access - return system tenant
				slog.Debug("CLI access granted via API key")
				// Note: For system/admin operations, you might want a special tenant
				// or handle this differently based on your security model
				return Tenant{Type: TenantTypeUser, ID: "system"}, true
			}
		}

		// Fall back to original extractor
		if originalExtractor != nil {
			return originalExtractor(r)
		}

		return Tenant{}, false
	}
	return config
}
