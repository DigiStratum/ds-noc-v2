# DS NOC v1 → v2 Migration Checklist

This document catalogs all features from ds-app-noc (v1) that need migration to ds-noc-v2.

**Source Repository:** ~/repos/digistratum/ds-app-noc
**Target Repository:** ~/repos/digistratum/ds-noc-v2
**Created:** 2026-04-01 (Issue #1932)

---

## Backend Endpoints

All endpoints should be scaffolded using GoTools before porting implementation.

| Method | Path | Handler | Status | Notes |
|--------|------|---------|--------|-------|
| GET | `/api/dashboard` | `DashboardHandler` | ⬜ TODO | Aggregated health status of all monitored services |
| GET | `/api/operations` | `OperationsHandler` | ⬜ TODO | Operational data (events, quick actions, maintenance windows) |
| GET | `/api/alerts` | `AlertsHandler` | ⬜ TODO | Service alerts with severity levels |
| GET | `/api/cloudwatch/metrics` | `cloudwatch.Handler` | ⬜ TODO | CloudWatch metrics integration |
| GET | `/api/session` | `GetSessionHandler` | ⬜ TODO | Session management (template-provided) |
| GET | `/api/me` | `GetCurrentUserHandler` | ⬜ TODO | Current user info (template-provided) |
| GET | `/api/tenant` | `GetCurrentTenantHandler` | ⬜ TODO | Current tenant info (template-provided) |
| GET | `/api/theme` | `theme.Handler` | ⬜ TODO | Tenant theme endpoint |
| GET | `/api/flags/evaluate` | `featureflags.EvaluateHandler` | ⬜ TODO | Feature flags evaluation |
| GET | `/api/flags` | `featureflags.ListHandler` | ⬜ TODO | List feature flags (admin) |
| PUT | `/api/flags/` | `featureflags.UpdateHandler` | ⬜ TODO | Update feature flag (admin) |
| DELETE | `/api/flags/` | `featureflags.DeleteHandler` | ⬜ TODO | Delete feature flag (admin) |
| GET | `/health` | `health.Handler` | ⬜ TODO | Health check endpoint |

### Auth Endpoints (Template-Provided)
| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| GET | `/api/auth/login` | `auth.LoginHandler` | DSAccount OAuth initiation |
| GET | `/api/auth/callback` | `auth.CallbackHandler` | OAuth callback |
| GET | `/api/auth/logout` | `auth.LogoutHandler` | Session termination |

---

## Backend Models

### Dashboard Models (`internal/api/dashboard.go`)

```go
// ServiceHealth - health status of a monitored service
type ServiceHealth struct {
    Status         string                 `json:"status"`        // healthy, degraded, unhealthy
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

// HealthCheck - individual health check result
type HealthCheck struct {
    Status    string `json:"status"`
    LatencyMs int    `json:"latencyMs,omitempty"`
    Message   string `json:"message,omitempty"`
}

// MemoryStats - memory utilization metrics
type MemoryStats struct {
    HeapUsedMB  float64 `json:"heapUsedMB"`
    HeapTotalMB float64 `json:"heapTotalMB"`
    RSSMB       float64 `json:"rssMB"`
    PercentUsed float64 `json:"percentUsed"`
}

// CPUStats - CPU utilization metrics
type CPUStats struct {
    LoadAverage [3]float64 `json:"loadAverage"`
    PercentUsed float64    `json:"percentUsed"`
}

// ConnectionStats - connection pool status
type ConnectionStats struct {
    Database *DBConnStats   `json:"database,omitempty"`
    HTTP     *HTTPConnStats `json:"http,omitempty"`
}

// DBConnStats - database connection pool stats
type DBConnStats struct {
    Active int `json:"active"`
    Idle   int `json:"idle"`
    Max    int `json:"max"`
}

// HTTPConnStats - HTTP connection stats
type HTTPConnStats struct {
    Active  int `json:"active"`
    Pending int `json:"pending"`
}

// DashboardState - response for /api/dashboard
type DashboardState struct {
    Services      map[string]*ServiceHealth `json:"services"`
    LastUpdated   string                    `json:"lastUpdated"`
    OverallStatus string                    `json:"overallStatus"` // healthy, degraded, unhealthy
}

// ServiceConfig - monitored service configuration
type ServiceConfig struct {
    ID             string `json:"id"`
    Name           string `json:"name"`
    URL            string `json:"url"`
    HealthEndpoint string `json:"healthEndpoint"`
    Critical       bool   `json:"criticalService"`
}
```

### Alert Models (`internal/api/alerts.go`)

```go
// Alert - service alert
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

// AlertsResponse - response for /api/alerts
type AlertsResponse struct {
    Alerts []Alert `json:"alerts"`
    Count  int     `json:"count"`
    Since  string  `json:"since"`
}
```

### Operations Models (`internal/api/operations.go`)

```go
// SystemEvent - operational event
type SystemEvent struct {
    ID        string `json:"id"`
    Timestamp string `json:"timestamp"`
    Type      string `json:"type"`      // deployment, alert, maintenance, config_change
    Severity  string `json:"severity"`  // info, warning, error
    Service   string `json:"service"`
    Message   string `json:"message"`
    User      string `json:"user,omitempty"`
}

// QuickAction - available operational action
type QuickAction struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Icon        string `json:"icon"`
    Enabled     bool   `json:"enabled"`
}

// MaintenanceWindow - scheduled maintenance period
type MaintenanceWindow struct {
    ID          string `json:"id"`
    Service     string `json:"service"`
    StartTime   string `json:"startTime"`
    EndTime     string `json:"endTime"`
    Description string `json:"description"`
}

// SystemLoad - current system load metrics
type SystemLoad struct {
    RequestsPerMinute int     `json:"requestsPerMinute"`
    ActiveConnections int     `json:"activeConnections"`
    QueuedJobs        int     `json:"queuedJobs"`
    ErrorRate         float64 `json:"errorRate"`
}

// OperationsData - response for /api/operations
type OperationsData struct {
    Events                     []SystemEvent       `json:"events"`
    QuickActions               []QuickAction       `json:"quickActions"`
    ScheduleMaintenanceWindows []MaintenanceWindow `json:"scheduleMaintenanceWindows"`
    SystemLoad                 SystemLoad          `json:"systemLoad"`
}
```

### CloudWatch Models (`internal/cloudwatch/handler.go`)

```go
// MetricDatapoint - single data point
type MetricDatapoint struct {
    Timestamp string  `json:"timestamp"`
    Value     float64 `json:"value"`
    Unit      string  `json:"unit"`
}

// MetricStatistics - computed statistics
type MetricStatistics struct {
    Average float64 `json:"average"`
    Maximum float64 `json:"maximum"`
    Minimum float64 `json:"minimum"`
    Sum     float64 `json:"sum"`
}

// MetricResult - single metric's data
type MetricResult struct {
    MetricName string            `json:"metricName"`
    Namespace  string            `json:"namespace"`
    Dimensions map[string]string `json:"dimensions"`
    Unit       string            `json:"unit"`
    Datapoints []MetricDatapoint `json:"datapoints"`
    Statistics MetricStatistics  `json:"statistics"`
}

// MetricsResponse - API response format
type MetricsResponse struct {
    Metrics   []MetricResult `json:"metrics"`
    Period    string         `json:"period"`
    StartTime string         `json:"startTime"`
    EndTime   string         `json:"endTime"`
}
```

---

## Frontend Pages

| Page | File | Status | Description |
|------|------|--------|-------------|
| NOC Dashboard | `pages/NocDashboard.tsx` | ⬜ TODO | Main dashboard with service health grid |
| Automation | `pages/Automation.tsx` | ⬜ TODO | CI/CD and automation status |
| Dashboard | `pages/Dashboard.tsx` | ⬜ TODO | General dashboard (may merge with NocDashboard) |
| Home | `pages/Home.tsx` | ⬜ TODO | Landing page |
| Settings | `pages/Settings.tsx` | ⬜ TODO | User settings page |

---

## Frontend Components

### NOC-Specific Components

| Component | File | Status | Description |
|-----------|------|--------|-------------|
| OverviewPanel | `components/OverviewPanel.tsx` | ⬜ TODO | System overview with key metrics |
| AlertsPanel | `components/AlertsPanel.tsx` | ⬜ TODO | Active alerts by severity |
| CloudWatchPanel | `components/CloudWatchPanel.tsx` | ⬜ TODO | CloudWatch metrics visualization |
| OperationsPanel | `components/OperationsPanel.tsx` | ⬜ TODO | Operations data (events, actions, maintenance) |
| ServiceCard | `components/ServiceCard.tsx` | ⬜ TODO | Compact service health card |
| ServiceDetail | `components/ServiceDetail.tsx` | ⬜ TODO | Expanded service detail modal |
| ResponseTimeChart | `components/ResponseTimeChart.tsx` | ⬜ TODO | Response time trend chart |
| AutomationDashboard | `components/AutomationDashboard.tsx` | ⬜ TODO | CI/CD pipeline status |

### Shell/Layout Components (Template-Provided)

| Component | File | Notes |
|-----------|------|-------|
| DeveloperAppShell | `components/DeveloperAppShell.tsx` | Use AppShell from template |
| DeveloperHeader | `components/DeveloperHeader.tsx` | Use Header from template |
| DeveloperFooter | `components/DeveloperFooter.tsx` | Use Footer from template |
| ErrorBoundary | `components/ErrorBoundary.tsx` | Template-provided |
| CookieConsent | `components/CookieConsent.tsx` | Template-provided |
| PreferencesModal | `components/PreferencesModal.tsx` | Template-provided |
| FeatureFlag | `components/FeatureFlag.tsx` | Template-provided |
| AdSlot | `components/AdSlot.tsx` | Optional - tenant ad integration |

---

## Frontend Hooks

| Hook | File | Status | Description |
|------|------|--------|-------------|
| useDashboard | `hooks/useDashboard.ts` | ⬜ TODO | Dashboard state and polling (10s interval) |
| useAutomation | `hooks/useAutomation.ts` | ⬜ TODO | Automation/DSKanban stats (30s interval) |
| useConsent | `hooks/useConsent.ts` | ⬜ TODO | Cookie consent management |
| useTenantTheme | `hooks/useTenantTheme.ts` | ⬜ TODO | Tenant theming |
| useAuth | `hooks/useAuth.tsx` | Template | Authentication (template-provided) |
| useFeatureFlags | `hooks/useFeatureFlags.tsx` | Template | Feature flags (template-provided) |
| useTheme | `hooks/useTheme.tsx` | Template | Theme toggle (template-provided) |

---

## Frontend Types (`types.ts`)

Key types to port:

```typescript
// Dashboard types
interface HealthCheck { status, latencyMs, message }
interface MemoryStats { heapUsedMB, heapTotalMB, rssMB, percentUsed }
interface CpuStats { loadAverage, percentUsed }
interface ConnectionStats { database, http }
interface ServiceHealth { status, version, uptime, timestamp, service, environment, checks, memory, cpu, connections, responseTimeMs }
interface ServiceConfig { id, name, url, healthEndpoint, criticalService }
interface DashboardState { services, lastUpdated, overallStatus }
interface HistoricalDataPoint { timestamp, responseTimeMs, status, errorRate }
interface ServiceHistory { serviceId, dataPoints }

// Auth types (template-provided)
interface User, Session, Tenant, TenantInfo, AppInfo, AuthContext

// Theme types (template-provided)
type Theme = 'light' | 'dark' | 'system'
interface ThemeContext
```

---

## API Client (`api/`)

| File | Status | Description |
|------|--------|-------------|
| `client.ts` | ⬜ TODO | Base API client wrapper |
| `dskanban.ts` | ⬜ TODO | DSKanban automation API client |
| `index.ts` | ⬜ TODO | API exports |

---

## Backend Internal Packages

| Package | Path | Status | Description |
|---------|------|--------|-------------|
| api | `internal/api/` | ⬜ TODO | API handlers (dashboard, operations, alerts) |
| auth | `internal/auth/` | Template | OAuth handlers (template-provided) |
| cloudwatch | `internal/cloudwatch/` | ⬜ TODO | CloudWatch integration |
| health | `internal/health/` | ⬜ TODO | Health check system |
| middleware | `internal/middleware/` | Template | HTTP middleware (mostly template) |
| session | `internal/session/` | Template | Session management (template) |
| theme | `internal/theme/` | ⬜ TODO | Tenant theming |
| featureflags | `internal/featureflags/` | ⬜ TODO | Feature flag system |
| dynamo | `internal/dynamo/` | Template | DynamoDB repository (template) |
| models | `internal/models/` | ⬜ TODO | Shared model definitions |

---

## Migration Order (Recommended)

### Phase 1: Core Infrastructure
1. ⬜ Scaffold health endpoint via GoTools
2. ⬜ Scaffold session/auth endpoints (template patterns)
3. ⬜ Port core middleware

### Phase 2: Dashboard Backend
1. ⬜ Scaffold `/api/dashboard` endpoint
2. ⬜ Port ServiceHealth, DashboardState models
3. ⬜ Port parallel health checking logic
4. ⬜ Scaffold `/api/alerts` endpoint
5. ⬜ Port Alert models and handler
6. ⬜ Scaffold `/api/operations` endpoint
7. ⬜ Port Operations models and handler

### Phase 3: CloudWatch Integration
1. ⬜ Scaffold `/api/cloudwatch/metrics` endpoint
2. ⬜ Port CloudWatch client and models
3. ⬜ Configure AWS SDK integration

### Phase 4: Frontend Pages
1. ⬜ Create NocDashboard page
2. ⬜ Port useDashboard hook
3. ⬜ Port ServiceCard component
4. ⬜ Port ServiceDetail component
5. ⬜ Port OverviewPanel component
6. ⬜ Port AlertsPanel component
7. ⬜ Port OperationsPanel component
8. ⬜ Port CloudWatchPanel component
9. ⬜ Port ResponseTimeChart component

### Phase 5: Automation Features
1. ⬜ Create Automation page
2. ⬜ Port useAutomation hook
3. ⬜ Port AutomationDashboard component
4. ⬜ Port DSKanban API client

### Phase 6: Testing & Documentation
1. ⬜ Add E2E tests for dashboard
2. ⬜ Add E2E tests for alerts
3. ⬜ Add E2E tests for CloudWatch
4. ⬜ Update REQUIREMENTS.md with @implements markers
5. ⬜ Verify CI/CD deployment

---

## Monitored Services Configuration

Default services monitored in v1 (hardcoded, should move to config/DynamoDB):

```go
var monitoredServices = []ServiceConfig{
    {ID: "dsaccount", Name: "DS Account", URL: "https://account.digistratum.com", HealthEndpoint: "/api/health", Critical: true},
    {ID: "dskanban", Name: "DS Projects", URL: "https://projects.digistratum.com", HealthEndpoint: "/api/health", Critical: false},
    {ID: "developer", Name: "DS Developer", URL: "https://developer.digistratum.com", HealthEndpoint: "/api/health", Critical: false},
}
```

**Improvement for v2:** Move service configuration to DynamoDB or environment config for runtime flexibility.

---

## Key Configuration Values

| Setting | Value | Source |
|---------|-------|--------|
| Dashboard poll interval | 10s | `useDashboard.ts` |
| Automation refresh interval | 30s | `useAutomation.ts` |
| Health check timeout | 5s | `dashboard.go` |
| Max history points | 60 | `useDashboard.ts` |

---

## Notes

- **Template preservation:** Do not modify shell/layout components directly. Use AppShell interface for menu injection.
- **GoTools mandatory:** All endpoint/model scaffolding must use GoTools commands.
- **REQUIREMENTS.md:** Each ported feature must have corresponding @implements marker.
- **Test coverage:** Add E2E tests with @covers markers for each feature.

---

*Last updated: 2026-04-01 by Issue #1932*
