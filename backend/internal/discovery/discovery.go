// Package discovery provides HAL-compliant API discovery.
package discovery

import (
	"encoding/json"
	"net/http"
	"os"
)

// ContentTypeHALJSON is the MIME type for HAL+JSON
const ContentTypeHALJSON = "application/hal+json"

// HALLink represents a HAL hypermedia link
type HALLink struct {
	Href string `json:"href"`
}

// HALCurie represents a HAL curie for compact URIs
type HALCurie struct {
	Name      string `json:"name"`
	Href      string `json:"href"`
	Templated bool   `json:"templated"`
}

// DiscoveryResponse is the HAL discovery response
type DiscoveryResponse struct {
	Service string                 `json:"service"`
	Version string                 `json:"version"`
	Links   map[string]interface{} `json:"_links"`
}

// AppLinksFunc returns app-specific HAL links to merge into discovery
type AppLinksFunc func() map[string]interface{}

// Version returns the application version from env or default
func Version() string {
	if v := os.Getenv("APP_VERSION"); v != "" {
		return v
	}
	return "1.0.0"
}

// ServiceName returns the service name from env or default
func ServiceName() string {
	if s := os.Getenv("SERVICE_NAME"); s != "" {
		return s
	}
	return "ds-app-template"
}

// BuildDiscoveryResponse creates a HAL-compliant discovery response
func BuildDiscoveryResponse() DiscoveryResponse {
	return DiscoveryResponse{
		Service: ServiceName(),
		Version: Version(),
		Links: map[string]interface{}{
			"self":              HALLink{Href: "/api/discovery"},
			"curies":           []HALCurie{{Name: "ds", Href: "https://developer.digistratum.com/docs/rels/{rel}", Templated: true}},
			"ds:health":        HALLink{Href: "/api/health"},
			"ds:session":       HALLink{Href: "/api/session"},
			"ds:theme":         HALLink{Href: "/api/theme"},
			"ds:auth-login":    HALLink{Href: "/api/auth/login"},
			"ds:auth-logout":   HALLink{Href: "/api/auth/logout"},
			"ds:me":            HALLink{Href: "/api/me"},
			"ds:tenant":        HALLink{Href: "/api/tenant"},
			"ds:flags-evaluate": HALLink{Href: "/api/flags/evaluate"},
			"ds:flags":         HALLink{Href: "/api/flags"},
			"ds:components":    HALLink{Href: "/api/components"},
		},
	}
}

// WriteHALJSON writes a HAL+JSON response
func WriteHALJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", ContentTypeHALJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Handler returns an http.HandlerFunc for the discovery endpoint.
// It accepts an optional AppLinksFunc to merge app-specific links.
func Handler(appLinks AppLinksFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			w.Header().Set("Content-Type", ContentTypeHALJSON)
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
			return
		}

		resp := BuildDiscoveryResponse()

		// Merge app-specific links if provided
		if appLinks != nil {
			for k, v := range appLinks() {
				resp.Links[k] = v
			}
		}

		WriteHALJSON(w, http.StatusOK, resp)
	}
}
