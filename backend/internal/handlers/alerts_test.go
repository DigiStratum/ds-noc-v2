package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAlerts(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantStatus int
		wantFields []string
	}{
		{
			name:       "default parameters",
			url:        "/api/alerts",
			wantStatus: http.StatusOK,
			wantFields: []string{"alerts", "count", "since", "_links"},
		},
		{
			name:       "with limit parameter",
			url:        "/api/alerts?limit=10",
			wantStatus: http.StatusOK,
			wantFields: []string{"alerts", "count", "since"},
		},
		{
			name:       "with hours parameter",
			url:        "/api/alerts?hours=48",
			wantStatus: http.StatusOK,
			wantFields: []string{"alerts", "count", "since"},
		},
		{
			name:       "with both parameters",
			url:        "/api/alerts?limit=5&hours=12",
			wantStatus: http.StatusOK,
			wantFields: []string{"alerts", "count", "since"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			ListAlerts(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			for _, field := range tt.wantFields {
				if _, ok := response[field]; !ok {
					t.Errorf("expected field %q in response", field)
				}
			}
		})
	}
}

func TestListAlertsResponseStructure(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/alerts", nil)
	w := httptest.NewRecorder()

	ListAlerts(w, req)

	var response AlertsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify alerts is an array (even if empty)
	if response.Alerts == nil {
		t.Error("expected alerts to be an array, got nil")
	}

	// Verify count matches alerts length
	if response.Count != len(response.Alerts) {
		t.Errorf("count %d does not match alerts length %d", response.Count, len(response.Alerts))
	}

	// Verify since is a valid timestamp
	if response.Since == "" {
		t.Error("expected since timestamp to be set")
	}

	// Verify HAL links
	if response.Links == nil {
		t.Error("expected _links to be present")
	} else {
		if _, ok := response.Links["self"]; !ok {
			t.Error("expected self link in _links")
		}
	}
}

func TestListAlertsInvalidParameters(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "invalid limit (non-numeric)",
			url:  "/api/alerts?limit=abc",
		},
		{
			name: "invalid limit (negative)",
			url:  "/api/alerts?limit=-5",
		},
		{
			name: "invalid hours (non-numeric)",
			url:  "/api/alerts?hours=xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			ListAlerts(w, req)

			// Should still return 200 with defaults
			if w.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}
