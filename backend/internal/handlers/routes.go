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
}

// RegisterDiscoveryLinks returns HAL links for app-specific endpoints.
// Called from discovery handler to include app routes in /api/discovery.
func RegisterDiscoveryLinks() map[string]interface{} {
	links := make(map[string]interface{})
	// App links are registered here by add-endpoint.sh
	// Example:
	// links["items"] = map[string]string{"href": "/api/items"}
	return links
}
