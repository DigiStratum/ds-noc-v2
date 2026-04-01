package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
)

// Alert represents a service alert
type Alert struct {
	ID             string `json:"id"`
	ServiceID      string `json:"serviceId"`
	ServiceName    string `json:"serviceName"`
	Timestamp      string `json:"timestamp"`
	Type           string `json:"type"`           // recovery, outage, degradation, change
	Severity       string `json:"severity"`       // critical, warning, info
	PreviousStatus string `json:"previousStatus"`
	CurrentStatus  string `json:"currentStatus"`
	Message        string `json:"message"`
	LatencyMs      int    `json:"latencyMs,omitempty"`
}

// AlertsResponse is the response for GET /api/alerts
type AlertsResponse struct {
	Alerts []Alert `json:"alerts"`
	Count  int     `json:"count"`
	Since  string  `json:"since"`
	Links  map[string]interface{} `json:"_links"`
}

// ListAlerts handles GET /api/alerts
// Returns recent alerts for monitored services
//
// Query parameters:
//   - limit: maximum number of alerts to return (default: 20)
//   - hours: lookback period in hours (default: 24)
func ListAlerts(w http.ResponseWriter, r *http.Request) {
	// Set txnlog data type for transaction logging
	middleware.SetTxnLogDataType(r.Context(), "Alert")

	// Parse query parameters
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	hours := 24
	if hoursStr := r.URL.Query().Get("hours"); hoursStr != "" {
		if parsed, err := strconv.Atoi(hoursStr); err == nil && parsed > 0 && parsed <= 168 {
			hours = parsed
		}
	}

	now := time.Now().UTC()
	since := now.Add(-time.Duration(hours) * time.Hour)

	// TODO: In production, fetch from CloudWatch Alarms, SNS, or DynamoDB
	// For now, return empty alerts (healthy system)
	alerts := []Alert{}

	// Respect limit (already empty, but good practice)
	if len(alerts) > limit {
		alerts = alerts[:limit]
	}

	response := AlertsResponse{
		Alerts: alerts,
		Count:  len(alerts),
		Since:  since.Format(time.RFC3339),
		Links: map[string]interface{}{
			"self": map[string]string{"href": "/api/alerts"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
