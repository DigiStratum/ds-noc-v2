// Package handlers contains app-specific API handlers.
// [FR-THEME-004, FR-THEME-005, FR-API-005]
package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/hal"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/session"
)

// ThemeConfig represents the theme configuration returned to clients.
// This structure allows tenant-specific branding and CSS customization.
type ThemeConfig struct {
	// CSSVars contains CSS custom property overrides
	// e.g., {"--ds-primary": "#ff6600", "--ds-accent": "#00cc66"}
	CSSVars map[string]string `json:"cssVars,omitempty"`

	// LogoURL is the URL for a custom logo (null = use default)
	LogoURL *string `json:"logoUrl,omitempty"`

	// LogoAlt is alt text for the logo
	LogoAlt string `json:"logoAlt,omitempty"`

	// FaviconURL is the URL for a custom favicon
	FaviconURL *string `json:"faviconUrl,omitempty"`
}

// ListThemes handles GET /api/theme
// Returns the theme configuration for the current session's tenant.
// If no custom theme is configured, returns an empty config (defaults apply).
//
// Response: HAL+JSON with ThemeConfig
// - 200: Theme config (may be empty for default theme)
func ListThemes(w http.ResponseWriter, r *http.Request) {
	// Set txnlog data type for transaction logging
	middleware.SetTxnLogDataType(r.Context(), "Theme")

	// Get tenant from session (if available)
	sess := session.GetSession(r.Context())
	tenantID := ""
	if sess != nil {
		tenantID = sess.TenantID
	}

	// Get theme config for tenant
	config := getThemeForTenant(tenantID)

	// Set cache header - theme doesn't change often
	w.Header().Set("Cache-Control", "private, max-age=300") // 5 minutes

	// Build HAL response
	response := hal.NewBuilder().
		Self("/api/theme").
		Data(config).
		Build()

	hal.WriteResource(w, http.StatusOK, response)
}

// getThemeForTenant retrieves the theme configuration for a tenant.
// Currently returns default empty config; will be replaced with DynamoDB lookup.
func getThemeForTenant(tenantID string) ThemeConfig {
	// TODO: Look up tenant theme from DynamoDB
	// For now, return empty config (frontend will use CSS defaults)
	//
	// Future implementation:
	// 1. Query DynamoDB themes table by tenant_id
	// 2. Check for user-level overrides (if authenticated)
	// 3. Merge tenant + user configs
	// 4. Cache result with TTL
	//
	// Placeholder for future DynamoDB integration:
	//
	// type ThemeRecord struct {
	//     TenantID   string            `dynamodbav:"tenant_id"`
	//     CSSVars    map[string]string `dynamodbav:"css_vars"`
	//     LogoURL    string            `dynamodbav:"logo_url"`
	//     FaviconURL string            `dynamodbav:"favicon_url"`
	//     UpdatedAt  time.Time         `dynamodbav:"updated_at"`
	// }
	//
	// Example query:
	// result, err := db.GetItem(&dynamodb.GetItemInput{
	//     TableName: aws.String("ds-themes"),
	//     Key: map[string]*dynamodb.AttributeValue{
	//         "tenant_id": {S: aws.String(tenantID)},
	//     },
	// })

	_ = tenantID // Unused for now

	// Return empty config - frontend CSS defaults will apply
	return ThemeConfig{}
}

// ThemePrefsFromCookie extracts theme preferences from ds_prefs cookie.
// Per DIGISTRATUM.md, preferences are client-side in ds_prefs cookie.
// The cookie value may be URL-encoded JSON.
func ThemePrefsFromCookie(r *http.Request) (map[string]string, error) {
	cookie, err := r.Cookie("ds_prefs")
	if err != nil {
		// No prefs cookie - return empty map, not an error
		return make(map[string]string), nil
	}

	value := cookie.Value

	// Try URL decoding first (browser may encode JSON characters)
	if decoded, err := url.QueryUnescape(value); err == nil {
		value = decoded
	}

	var prefs map[string]string
	if err := json.Unmarshal([]byte(value), &prefs); err != nil {
		return make(map[string]string), nil
	}

	return prefs, nil
}
