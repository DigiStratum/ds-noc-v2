package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListOperations(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/operations", nil)
	w := httptest.NewRecorder()

	ListOperations(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify Content-Type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/hal+json" && contentType != "application/json" {
		t.Errorf("expected JSON content type, got %s", contentType)
	}

	// Parse response as generic map (HAL flattens data)
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify _links present
	if _, ok := response["_links"]; !ok {
		t.Error("expected _links in response")
	}

	// Check required fields exist (HAL flattens OperationsData fields)
	requiredFields := []string{"events", "quickActions", "scheduleMaintenanceWindows", "systemLoad"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("expected %s field in response", field)
		}
	}
}

func TestListOperations_ReturnsEvents(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/operations", nil)
	w := httptest.NewRecorder()

	ListOperations(w, req)

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	events, ok := response["events"].([]interface{})
	if !ok {
		t.Fatal("expected events to be an array")
	}

	if len(events) == 0 {
		t.Error("expected at least one event in response")
	}

	// Verify event structure
	firstEvent := events[0].(map[string]interface{})
	eventFields := []string{"id", "timestamp", "type", "service", "message"}
	for _, field := range eventFields {
		if _, exists := firstEvent[field]; !exists {
			t.Errorf("expected %s field in event", field)
		}
	}
}

func TestListOperations_ReturnsQuickActions(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/operations", nil)
	w := httptest.NewRecorder()

	ListOperations(w, req)

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	actions, ok := response["quickActions"].([]interface{})
	if !ok {
		t.Fatal("expected quickActions to be an array")
	}

	if len(actions) == 0 {
		t.Error("expected at least one quick action in response")
	}

	// Verify action structure
	firstAction := actions[0].(map[string]interface{})
	actionFields := []string{"id", "name", "description"}
	for _, field := range actionFields {
		if _, exists := firstAction[field]; !exists {
			t.Errorf("expected %s field in quick action", field)
		}
	}
}

func TestListOperations_ReturnsSystemLoad(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/operations", nil)
	w := httptest.NewRecorder()

	ListOperations(w, req)

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	systemLoad, ok := response["systemLoad"].(map[string]interface{})
	if !ok {
		t.Fatal("expected systemLoad to be an object")
	}

	// Verify system load fields
	loadFields := []string{"requestsPerMinute", "activeConnections", "queuedJobs", "errorRate"}
	for _, field := range loadFields {
		if _, exists := systemLoad[field]; !exists {
			t.Errorf("expected %s field in systemLoad", field)
		}
	}
}
