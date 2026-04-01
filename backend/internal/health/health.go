// Package health provides shallow and deep health check capabilities.
//
// Shallow health checks are fast, unauthenticated probes for load balancers.
// Deep health checks require authentication and check all dependencies.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Status represents the health status of a service or dependency.
type Status string

const (
	StatusUp       Status = "up"
	StatusDegraded Status = "degraded"
	StatusDown     Status = "down"
)

// Depth represents the depth of health check to perform.
type Depth string

const (
	DepthShallow Depth = "shallow"
	DepthDeep    Depth = "deep"
)

// DependencyConfig defines a dependency to check.
type DependencyConfig struct {
	Name        string        `json:"name"`
	URL         string        `json:"url"`
	TimeoutMs   int           `json:"timeout_ms"`
	Critical    bool          `json:"critical"`    // If critical dependency is down, overall status is down
	Description string        `json:"description"` // Human-readable description
}

// Config holds the health check configuration.
type Config struct {
	Dependencies    []DependencyConfig `json:"dependencies"`
	DefaultTimeout  time.Duration      `json:"-"`
}

// DependencyResult represents the health check result for a single dependency.
type DependencyResult struct {
	Name        string `json:"name"`
	Status      Status `json:"status"`
	LatencyMs   int64  `json:"latency_ms"`
	Message     string `json:"message,omitempty"`
	Critical    bool   `json:"critical"`
	Description string `json:"description,omitempty"`
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status       Status              `json:"status"`
	Timestamp    string              `json:"timestamp"`
	Version      string              `json:"version"`
	Uptime       string              `json:"uptime,omitempty"`       // Only in deep check
	UptimePct    *float64            `json:"uptime_pct,omitempty"`   // Only in deep check
	Dependencies []DependencyResult  `json:"dependencies,omitempty"` // Only in deep check
	Latency      *LatencyMetrics     `json:"latency,omitempty"`      // Only in deep check
}

// LatencyMetrics provides latency statistics.
type LatencyMetrics struct {
	TotalMs int64 `json:"total_ms"` // Total time to complete all checks
	MinMs   int64 `json:"min_ms"`   // Minimum dependency latency
	MaxMs   int64 `json:"max_ms"`   // Maximum dependency latency
	AvgMs   int64 `json:"avg_ms"`   // Average dependency latency
}

var (
	startTime = time.Now()
	config    *Config
	configMu  sync.RWMutex
)

// GetConfig returns the current health check configuration.
func GetConfig() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	
	if config == nil {
		return loadConfigFromEnv()
	}
	return config
}

// SetConfig sets the health check configuration (mainly for testing).
func SetConfig(c *Config) {
	configMu.Lock()
	defer configMu.Unlock()
	config = c
}

// loadConfigFromEnv loads configuration from environment variables.
// HEALTH_DEPENDENCIES format: name|url|timeout_ms|critical,name2|url2|timeout_ms2|critical2
// Example: dsaccount|https://account.digistratum.com/health|5000|true,dskanban|https://kanban.digistratum.com/health|3000|false
func loadConfigFromEnv() *Config {
	cfg := &Config{
		DefaultTimeout: 5 * time.Second,
	}
	
	depsEnv := os.Getenv("HEALTH_DEPENDENCIES")
	if depsEnv == "" {
		return cfg
	}
	
	deps := strings.Split(depsEnv, ",")
	for _, dep := range deps {
		parts := strings.Split(strings.TrimSpace(dep), "|")
		if len(parts) < 2 {
			continue
		}
		
		d := DependencyConfig{
			Name:      parts[0],
			URL:       parts[1],
			TimeoutMs: 5000, // default
			Critical:  false,
		}
		
		if len(parts) > 2 {
			if timeout, err := parseInt(parts[2]); err == nil {
				d.TimeoutMs = timeout
			}
		}
		
		if len(parts) > 3 {
			d.Critical = parts[3] == "true"
		}
		
		if len(parts) > 4 {
			d.Description = parts[4]
		}
		
		cfg.Dependencies = append(cfg.Dependencies, d)
	}
	
	return cfg
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// CheckDependency performs a health check on a single dependency.
func CheckDependency(ctx context.Context, dep DependencyConfig) DependencyResult {
	result := DependencyResult{
		Name:        dep.Name,
		Critical:    dep.Critical,
		Description: dep.Description,
	}
	
	timeout := time.Duration(dep.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	start := time.Now()
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dep.URL, nil)
	if err != nil {
		result.Status = StatusDown
		result.Message = fmt.Sprintf("failed to create request: %v", err)
		result.LatencyMs = time.Since(start).Milliseconds()
		return result
	}
	
	// Use a custom client with reasonable defaults
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	
	resp, err := client.Do(req)
	result.LatencyMs = time.Since(start).Milliseconds()
	
	if err != nil {
		result.Status = StatusDown
		if ctx.Err() == context.DeadlineExceeded {
			result.Message = fmt.Sprintf("timeout after %dms", dep.TimeoutMs)
		} else {
			result.Message = fmt.Sprintf("request failed: %v", err)
		}
		return result
	}
	defer func() { _ = resp.Body.Close() }()
	
	// Check status code
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		result.Status = StatusUp
	case resp.StatusCode >= 500:
		result.Status = StatusDown
		result.Message = fmt.Sprintf("server error: %d", resp.StatusCode)
	default:
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
	}
	
	return result
}

// CheckDependenciesParallel checks all dependencies in parallel.
func CheckDependenciesParallel(ctx context.Context, deps []DependencyConfig) []DependencyResult {
	if len(deps) == 0 {
		return nil
	}
	
	results := make([]DependencyResult, len(deps))
	var wg sync.WaitGroup
	
	for i, dep := range deps {
		wg.Add(1)
		go func(idx int, d DependencyConfig) {
			defer wg.Done()
			results[idx] = CheckDependency(ctx, d)
		}(i, dep)
	}
	
	wg.Wait()
	return results
}

// CalculateOverallStatus determines the overall status based on dependency results.
func CalculateOverallStatus(results []DependencyResult) Status {
	if len(results) == 0 {
		return StatusUp
	}
	
	hasDegraded := false
	
	for _, r := range results {
		if r.Status == StatusDown && r.Critical {
			return StatusDown
		}
		if r.Status == StatusDegraded || (r.Status == StatusDown && !r.Critical) {
			hasDegraded = true
		}
	}
	
	if hasDegraded {
		return StatusDegraded
	}
	
	return StatusUp
}

// CalculateLatencyMetrics computes latency statistics from dependency results.
func CalculateLatencyMetrics(results []DependencyResult, totalMs int64) *LatencyMetrics {
	if len(results) == 0 {
		return nil
	}
	
	metrics := &LatencyMetrics{
		TotalMs: totalMs,
		MinMs:   results[0].LatencyMs,
		MaxMs:   results[0].LatencyMs,
	}
	
	var sum int64
	for _, r := range results {
		sum += r.LatencyMs
		if r.LatencyMs < metrics.MinMs {
			metrics.MinMs = r.LatencyMs
		}
		if r.LatencyMs > metrics.MaxMs {
			metrics.MaxMs = r.LatencyMs
		}
	}
	
	metrics.AvgMs = sum / int64(len(results))
	return metrics
}

// GetUptime returns the service uptime as a duration string.
func GetUptime() string {
	return time.Since(startTime).Round(time.Second).String()
}

// GetUptimePercent returns a simulated uptime percentage.
// In production, this would be calculated from actual metrics/CloudWatch.
func GetUptimePercent() float64 {
	// For the boilerplate, return a simulated value.
	// Production implementation would query CloudWatch metrics:
	// - Calculate: (successful_requests / total_requests) * 100
	// - Or query availability metrics from the last 24h/7d
	return 99.9
}

// Version returns the application version.
func Version() string {
	v := os.Getenv("APP_VERSION")
	if v == "" {
		return "1.0.0"
	}
	return v
}

// ShallowCheck performs a quick health check without dependencies.
func ShallowCheck() *HealthResponse {
	return &HealthResponse{
		Status:    StatusUp,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   Version(),
	}
}

// DeepCheck performs a comprehensive health check including all dependencies.
func DeepCheck(ctx context.Context) *HealthResponse {
	cfg := GetConfig()
	
	start := time.Now()
	var depResults []DependencyResult
	
	if len(cfg.Dependencies) > 0 {
		depResults = CheckDependenciesParallel(ctx, cfg.Dependencies)
	}
	
	totalMs := time.Since(start).Milliseconds()
	uptimePct := GetUptimePercent()
	
	return &HealthResponse{
		Status:       CalculateOverallStatus(depResults),
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Version:      Version(),
		Uptime:       GetUptime(),
		UptimePct:    &uptimePct,
		Dependencies: depResults,
		Latency:      CalculateLatencyMetrics(depResults, totalMs),
	}
}

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
