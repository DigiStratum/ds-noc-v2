package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestListThemes_ReturnsOK tests the basic happy path
// @covers [FR-THEME-004, FR-THEME-005, FR-API-005]
func TestListThemes_ReturnsOK(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/theme", nil)
	w := httptest.NewRecorder()

	ListThemes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// TestListThemes_ReturnsHALJSON tests that response is HAL+JSON formatted
// @covers [FR-THEME-004]
func TestListThemes_ReturnsHALJSON(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/theme", nil)
	w := httptest.NewRecorder()

	ListThemes(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/hal+json" {
		t.Errorf("expected Content-Type application/hal+json, got %s", contentType)
	}

	// Verify _links.self is present (HAL standard)
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	links, ok := response["_links"].(map[string]interface{})
	if !ok {
		t.Fatal("expected _links in HAL response")
	}

	self, ok := links["self"].(map[string]interface{})
	if !ok {
		t.Fatal("expected _links.self in HAL response")
	}

	if href, ok := self["href"].(string); !ok || href != "/api/theme" {
		t.Errorf("expected _links.self.href to be /api/theme, got %v", self["href"])
	}
}

// TestListThemes_SetsCacheControl tests cache header is set
// @covers [FR-THEME-005]
func TestListThemes_SetsCacheControl(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/theme", nil)
	w := httptest.NewRecorder()

	ListThemes(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "private") {
		t.Errorf("expected Cache-Control to contain 'private', got %s", cacheControl)
	}
	if !strings.Contains(cacheControl, "max-age=300") {
		t.Errorf("expected Cache-Control to contain 'max-age=300', got %s", cacheControl)
	}
}

// TestListThemes_ReturnsEmptyDefaultConfig tests empty config for no tenant
// @covers [FR-THEME-004]
func TestListThemes_ReturnsEmptyDefaultConfig(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/theme", nil)
	w := httptest.NewRecorder()

	ListThemes(w, req)

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Empty theme config should not have cssVars, logoUrl, etc. (only _links)
	// The data should be an empty object when marshaled
	if cssVars, ok := response["cssVars"]; ok && cssVars != nil {
		t.Errorf("expected cssVars to be empty/nil for default theme, got %v", cssVars)
	}
}

// TestThemePrefsFromCookie_NoPrefs tests behavior when ds_prefs cookie is missing
func TestThemePrefsFromCookie_NoPrefs(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/theme", nil)

	prefs, err := ThemePrefsFromCookie(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if prefs == nil {
		t.Error("expected empty map, got nil")
	}

	if len(prefs) != 0 {
		t.Errorf("expected empty map, got %d items", len(prefs))
	}
}

// TestThemePrefsFromCookie_ValidPrefs tests parsing valid prefs cookie
// Note: Browser-set cookies will URL-encode special characters
func TestThemePrefsFromCookie_ValidPrefs(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/theme", nil)
	// URL-encoded JSON as would be sent by browser
	req.AddCookie(&http.Cookie{
		Name:  "ds_prefs",
		Value: "%7B%22theme%22%3A%22dark%22%2C%22accent%22%3A%22blue%22%7D", // {"theme":"dark","accent":"blue"}
	})

	prefs, err := ThemePrefsFromCookie(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if prefs["theme"] != "dark" {
		t.Errorf("expected theme=dark, got %s", prefs["theme"])
	}

	if prefs["accent"] != "blue" {
		t.Errorf("expected accent=blue, got %s", prefs["accent"])
	}
}

// TestThemePrefsFromCookie_InvalidJSON tests graceful handling of malformed prefs
func TestThemePrefsFromCookie_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/theme", nil)
	req.AddCookie(&http.Cookie{
		Name:  "ds_prefs",
		Value: "not-valid-json",
	})

	prefs, err := ThemePrefsFromCookie(req)
	if err != nil {
		t.Errorf("unexpected error for invalid JSON: %v", err)
	}

	if prefs == nil {
		t.Error("expected empty map on invalid JSON, got nil")
	}

	if len(prefs) != 0 {
		t.Errorf("expected empty map on invalid JSON, got %d items", len(prefs))
	}
}
