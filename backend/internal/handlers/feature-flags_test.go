package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListFeatureFlags(t *testing.T) {
	// Note: This test will fail in isolation since it requires DynamoDB
	// In real tests, we would mock the store
	t.Skip("Requires DynamoDB connection - integration test")
	
	req := httptest.NewRequest("GET", "/api/feature-flags", nil)
	w := httptest.NewRecorder()

	ListFeatureFlags(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response is HAL-compliant
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/hal+json" {
		t.Errorf("expected Content-Type application/hal+json, got %s", contentType)
	}
}

func TestPatchFeatureFlag(t *testing.T) {
	t.Skip("Requires DynamoDB connection - integration test")
}

func TestDeleteFeatureFlag(t *testing.T) {
	t.Skip("Requires DynamoDB connection - integration test")
}

func TestExtractFlagKey(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		prefix   string
		expected string
	}{
		{
			name:     "valid key",
			path:     "/api/feature-flags/my-flag",
			prefix:   "/api/feature-flags/",
			expected: "my-flag",
		},
		{
			name:     "key with trailing slash",
			path:     "/api/feature-flags/my-flag/",
			prefix:   "/api/feature-flags/",
			expected: "my-flag",
		},
		{
			name:     "empty key",
			path:     "/api/feature-flags/",
			prefix:   "/api/feature-flags/",
			expected: "",
		},
		{
			name:     "no match",
			path:     "/api/other/key",
			prefix:   "/api/feature-flags/",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractFlagKey(tc.path, tc.prefix)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
