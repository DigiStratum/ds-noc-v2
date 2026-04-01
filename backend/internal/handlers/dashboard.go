package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
)

// ServiceHealth represents the health status of a monitored service
type ServiceHealth struct {
	Status         string                 `json:"status"` // healthy, degraded, unhealthy
	Version        string                 `json:"version,omitempty"`
	Uptime         int                    `json:"uptime,omitempty"`
	Timestamp      string                 `json:"timestamp"`
	Service        string                 `json:"service,omitempty"`
	Environment    string                 `json:"environment,omitempty"`
	Checks         map[string]HealthCheck `json:"checks,omitempty"`
	Memory         *MemoryStats           `json:"memory,omitempty"`
	CPU            *CPUStats              `json:"cpu,omitempty"`
	Connections    *ConnectionStats       `json:"connections,omitempty"`
	ResponseTimeMs int                    `json:"responseTimeMs"`
}

// HealthCheck represents an individual health check result
type HealthCheck struct {
	Status    string `json:"status"`
	LatencyMs int    `json:"latencyMs,omitempty"`
	Message   string `json:"message,omitempty"`
}

// MemoryStats contains memory usage metrics
type MemoryStats struct {
	HeapUsedMB  float64 `json:"heapUsedMB"`
	HeapTotalMB float64 `json:"heapTotalMB"`
	RSSMB       float64 `json:"rssMB"`
	PercentUsed float64 `json:"percentUsed"`
}

// CPUStats contains CPU usage metrics
type CPUStats struct {
	LoadAverage [3]float64 `json:"loadAverage"`
	PercentUsed float64    `json:"percentUsed"`
}

// ConnectionStats contains connection pool metrics
type ConnectionStats struct {
	Database *DBConnStats   `json:"database,omitempty"`
	HTTP     *HTTPConnStats `json:"http,omitempty"`
}

// DBConnStats contains database connection pool metrics
type DBConnStats struct {
	Active int `json:"active"`
	Idle   int `json:"idle"`
	Max    int `json:"max"`
}

// HTTPConnStats contains HTTP connection pool metrics
type HTTPConnStats struct {
	Active  int `json:"active"`
	Pending int `json:"pending"`
}

// DashboardState is the response for GET /api/dashboard
type DashboardState struct {
	Services      map[string]*ServiceHealth `json:"services"`
	LastUpdated   string                    `json:"lastUpdated"`
	OverallStatus string                    `json:"overallStatus"` // healthy, degraded, unhealthy
}

// ServiceConfig defines a service to monitor
type ServiceConfig struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	URL            string `json:"url"`
	HealthEndpoint string `json:"healthEndpoint"`
	Critical       bool   `json:"criticalService"`
}

// monitoredServices - in production these would come from config/DynamoDB
var monitoredServices = []ServiceConfig{
	{ID: "dsaccount", Name: "DS Account", URL: "https://account.digistratum.com", HealthEndpoint: "/api/health", Critical: true},
	{ID: "dskanban", Name: "DS Projects", URL: "https://projects.digistratum.com", HealthEndpoint: "/api/health", Critical: false},
	{ID: "developer", Name: "DS Developer", URL: "https://developer.digistratum.com", HealthEndpoint: "/api/health", Critical: false},
}

// ListDashboards handles GET /api/dashboard
// Returns aggregated health status of all monitored services
func ListDashboards(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Set txnlog data type for transaction logging
	middleware.SetTxnLogDataType(ctx, "Dashboard")

	services := make(map[string]*ServiceHealth)
	var wg sync.WaitGroup
	var mu sync.Mutex

	client := &http.Client{Timeout: 5 * time.Second}

	for _, svc := range monitoredServices {
		wg.Add(1)
		go func(svc ServiceConfig) {
			defer wg.Done()

			start := time.Now()
			health := fetchServiceHealth(ctx, client, svc)
			health.ResponseTimeMs = int(time.Since(start).Milliseconds())

			mu.Lock()
			services[svc.ID] = health
			mu.Unlock()
		}(svc)
	}

	wg.Wait()

	// Determine overall status
	overallStatus := "healthy"
	for _, health := range services {
		if health == nil {
			overallStatus = "unhealthy"
			break
		}
		if health.Status == "unhealthy" {
			overallStatus = "unhealthy"
			break
		}
		if health.Status == "degraded" && overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}

	response := DashboardState{
		Services:      services,
		LastUpdated:   time.Now().UTC().Format(time.RFC3339),
		OverallStatus: overallStatus,
	}

	slog.Info("dashboard fetched", "services", len(services), "status", overallStatus)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func fetchServiceHealth(ctx context.Context, client *http.Client, svc ServiceConfig) *ServiceHealth {
	url := svc.URL + svc.HealthEndpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &ServiceHealth{
			Status:    "unhealthy",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Service:   svc.Name,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return &ServiceHealth{
			Status:    "unhealthy",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Service:   svc.Name,
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return &ServiceHealth{
			Status:    "degraded",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Service:   svc.Name,
		}
	}

	var health ServiceHealth
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		// Health endpoint returned 200 but bad JSON - still consider healthy
		return &ServiceHealth{
			Status:    "healthy",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Service:   svc.Name,
		}
	}

	health.Service = svc.Name
	if health.Status == "" {
		health.Status = "healthy"
	}
	if health.Timestamp == "" {
		health.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	return &health
}
