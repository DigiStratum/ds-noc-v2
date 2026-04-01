// Package handlers contains app-specific API handlers.
// This file is APP-OWNED and will not be overwritten by template updates.
package handlers

import "net/http"

// RegisterRoutes registers all app-specific routes.
// Called from main.go during server initialization.
//
// Use scripts/add-endpoint.sh to scaffold new handlers:
//   ./scripts/add-endpoint.sh GET /api/items ListItems
//
// The script will:
// 1. Create handler function in this package
// 2. Add route registration below
// 3. Add HAL link to discovery.go
func RegisterRoutes(mux *http.ServeMux) {
	// App routes are registered here by add-endpoint.sh
	// Example:
	// mux.HandleFunc("GET /api/items", ListItems)
	// mux.HandleFunc("POST /api/items", CreateItem)
	// mux.HandleFunc("GET /api/items/{id}", GetItem)

	// Theme endpoint - returns tenant-specific theme configuration
	mux.HandleFunc("GET /api/theme", ListThemes)

	// Dashboard endpoint - returns aggregated NOC service health
	mux.HandleFunc("GET /api/dashboard", ListDashboards)

	// Operations endpoint - returns operational data for NOC operations panel
	mux.HandleFunc("GET /api/operations", ListOperations)

	// Alerts endpoint - returns recent alerts for monitored services
	mux.HandleFunc("GET /api/alerts", ListAlerts)

	// Feature flags endpoints
	mux.HandleFunc("GET /api/feature-flags", ListFeatureFlags)
	mux.HandleFunc("GET /api/feature-flags/evaluate", EvaluateFeatureFlags)
	mux.HandleFunc("PATCH /api/feature-flags/{key}", PatchFeatureFlag)
	mux.HandleFunc("DELETE /api/feature-flags/{key}", DeleteFeatureFlag)
}

// RegisterDiscoveryLinks returns HAL links for app-specific endpoints.
// Called from discovery handler to include app routes in /api/discovery.
func RegisterDiscoveryLinks() map[string]interface{} {
	links := make(map[string]interface{})
	// App links are registered here by add-endpoint.sh
	// Example:
	// links["items"] = map[string]string{"href": "/api/items"}
		links["theme"] = map[string]string{"href": "/api/theme"}
		links["dashboard"] = map[string]string{"href": "/api/dashboard"}
		links["operations"] = map[string]string{"href": "/api/operations"}
		links["alerts"] = map[string]string{"href": "/api/alerts"}
		links["feature-flags"] = map[string]string{"href": "/api/feature-flags"}
		links["feature-flags:evaluate"] = map[string]string{"href": "/api/feature-flags/evaluate"}
	return links
}
