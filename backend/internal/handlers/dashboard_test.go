package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListDashboards_ReturnsExpectedStructure(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/dashboard", nil)
	w := httptest.NewRecorder()

	ListDashboards(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response DashboardState
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify required fields
	if response.Services == nil {
		t.Error("expected services map to be non-nil")
	}

	if response.LastUpdated == "" {
		t.Error("expected lastUpdated to be set")
	}

	if response.OverallStatus == "" {
		t.Error("expected overallStatus to be set")
	}

	// Verify overallStatus is a valid value
	validStatuses := map[string]bool{"healthy": true, "degraded": true, "unhealthy": true}
	if !validStatuses[response.OverallStatus] {
		t.Errorf("unexpected overallStatus: %s", response.OverallStatus)
	}
}

func TestListDashboards_ContentType(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/dashboard", nil)
	w := httptest.NewRecorder()

	ListDashboards(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestServiceHealth_StatusValues(t *testing.T) {
	// Verify the ServiceHealth type can represent all status values
	statuses := []string{"healthy", "degraded", "unhealthy"}
	for _, status := range statuses {
		health := ServiceHealth{Status: status}
		if health.Status != status {
			t.Errorf("expected status %s, got %s", status, health.Status)
		}
	}
}

func TestFetchServiceHealth_UnreachableService(t *testing.T) {
	// Test that unreachable services return unhealthy status
	svc := ServiceConfig{
		ID:             "test-service",
		Name:           "Test Service",
		URL:            "http://localhost:99999", // Unreachable port
		HealthEndpoint: "/health",
		Critical:       false,
	}

	client := &http.Client{}
	health := fetchServiceHealth(nil, client, svc)

	if health.Status != "unhealthy" {
		t.Errorf("expected unhealthy status for unreachable service, got %s", health.Status)
	}

	if health.Service != "Test Service" {
		t.Errorf("expected service name 'Test Service', got %s", health.Service)
	}
}
