package handlers

import (
	"net/http"
	"time"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/hal"
	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
)

// SystemEvent represents an operational event
type SystemEvent struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`     // deployment, alert, maintenance, config_change
	Severity  string `json:"severity"` // info, warning, error
	Service   string `json:"service"`
	Message   string `json:"message"`
	Status    string `json:"status"` // in_progress, completed, failed
	User      string `json:"user,omitempty"`
}

// QuickAction represents an available operational action
type QuickAction struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"`
	Service     string `json:"service,omitempty"`
	Dangerous   bool   `json:"dangerous,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// MaintenanceWindow represents a scheduled maintenance period
type MaintenanceWindow struct {
	ID          string `json:"id"`
	Service     string `json:"service"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	Description string `json:"description"`
}

// SystemLoad represents current system load metrics
type SystemLoad struct {
	RequestsPerMinute int     `json:"requestsPerMinute"`
	ActiveConnections int     `json:"activeConnections"`
	QueuedJobs        int     `json:"queuedJobs"`
	ErrorRate         float64 `json:"errorRate"`
}

// OperationsData is the response for /api/operations
type OperationsData struct {
	Events                     []SystemEvent       `json:"events"`
	QuickActions               []QuickAction       `json:"quickActions"`
	ScheduleMaintenanceWindows []MaintenanceWindow `json:"scheduleMaintenanceWindows"`
	SystemLoad                 SystemLoad          `json:"systemLoad"`
}

// ListOperations handles GET /api/operations
// Returns operational data for the NOC dashboard including events, quick actions,
// maintenance windows, and current system load metrics.
func ListOperations(w http.ResponseWriter, r *http.Request) {
	// Set txnlog data type for transaction logging
	middleware.SetTxnLogDataType(r.Context(), "Operation")

	// TODO: In production, fetch from CloudWatch Logs, EventBridge, etc.
	// For now, return mock data to make the dashboard functional
	now := time.Now().UTC()

	operationsData := OperationsData{
		Events: []SystemEvent{
			{
				ID:        "evt-001",
				Timestamp: now.Add(-5 * time.Minute).Format(time.RFC3339),
				Type:      "deployment",
				Severity:  "info",
				Service:   "DS Account",
				Message:   "Deployment completed successfully",
				Status:    "completed",
			},
			{
				ID:        "evt-002",
				Timestamp: now.Add(-15 * time.Minute).Format(time.RFC3339),
				Type:      "config_change",
				Severity:  "info",
				Service:   "DS Projects",
				Message:   "Feature flag updated: dark_mode_enabled",
				Status:    "completed",
				User:      "lucca",
			},
		},
		QuickActions: []QuickAction{
			{ID: "action-1", Name: "Clear Cache", Description: "Clear CloudFront cache for all distributions", Icon: "trash", Enabled: true},
			{ID: "action-2", Name: "Restart Lambda", Description: "Force cold start on Lambda functions", Icon: "refresh", Enabled: true},
			{ID: "action-3", Name: "Run Health Check", Description: "Trigger immediate health check on all services", Icon: "heart", Enabled: true},
		},
		ScheduleMaintenanceWindows: []MaintenanceWindow{},
		SystemLoad: SystemLoad{
			RequestsPerMinute: 42,
			ActiveConnections: 8,
			QueuedJobs:        0,
			ErrorRate:         0.02,
		},
	}

	response := hal.NewBuilder().
		Self("/api/operations").
		Data(operationsData).
		Build()

	hal.WriteResource(w, http.StatusOK, response)
}



