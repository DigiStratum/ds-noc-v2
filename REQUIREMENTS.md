# REQUIREMENTS.md - DS NOC v2

Requirements traceability document for ds-noc-v2 Network Operations Center application.

---

## Migration Checklist (from ds-app-noc v1)

This section documents all features from ds-app-noc that need to be migrated to ds-noc-v2.
**Source:** `~/repos/digistratum/ds-app-noc`

### Backend Endpoints

| Method | Path | Handler | Status | Notes |
|--------|------|---------|--------|-------|
| GET | `/api/dashboard` | `api.DashboardHandler` | TODO | Aggregated health status of all monitored services |
| GET | `/api/operations` | `api.OperationsHandler` | TODO | Operational data (events, quick actions, maintenance windows) |
| GET | `/api/alerts` | `api.AlertsHandler` | TODO | Service alerts by severity |
| GET | `/api/cloudwatch/metrics` | `cloudwatch.Handler` | TODO | CloudWatch metrics data (latency, errors, invocations) |

### Backend Models

#### Service Health Models (`internal/api/dashboard.go`)

| Model | Key Fields | Status |
|-------|------------|--------|
| `ServiceHealth` | status, version, uptime, timestamp, service, environment, checks, memory, cpu, connections, responseTimeMs | TODO |
| `HealthCheck` | status, latencyMs, message | TODO |
| `MemoryStats` | heapUsedMB, heapTotalMB, rssMB, percentUsed | TODO |
| `CPUStats` | loadAverage[3], percentUsed | TODO |
| `ConnectionStats` | database (DBConnStats), http (HTTPConnStats) | TODO |
| `DBConnStats` | active, idle, max | TODO |
| `HTTPConnStats` | active, pending | TODO |
| `DashboardState` | services (map[string]*ServiceHealth), lastUpdated, overallStatus | TODO |
| `ServiceConfig` | id, name, url, healthEndpoint, critical | TODO |

#### Operations Models (`internal/api/operations.go`)

| Model | Key Fields | Status |
|-------|------------|--------|
| `SystemEvent` | id, timestamp, type (deployment/alert/maintenance/config_change), severity, service, message, user | TODO |
| `QuickAction` | id, name, description, icon, enabled | TODO |
| `MaintenanceWindow` | id, service, startTime, endTime, description | TODO |
| `SystemLoad` | requestsPerMinute, activeConnections, queuedJobs, errorRate | TODO |
| `OperationsData` | events, quickActions, scheduleMaintenanceWindows, systemLoad | TODO |

#### Alert Models (`internal/api/alerts.go`)

| Model | Key Fields | Status |
|-------|------------|--------|
| `Alert` | id, serviceId, serviceName, timestamp, type (recovery/outage/degradation/change), severity (critical/warning/info), previousStatus, currentStatus, message, latencyMs | TODO |
| `AlertsResponse` | alerts, count, since | TODO |

#### CloudWatch Models (`internal/cloudwatch/handler.go`)

| Model | Key Fields | Status |
|-------|------------|--------|
| `MetricDatapoint` | timestamp, value, unit | TODO |
| `MetricStatistics` | average, maximum, minimum, sum | TODO |
| `MetricResult` | metricName, namespace, dimensions, unit, datapoints, statistics | TODO |
| `MetricsResponse` | metrics, period, startTime, endTime | TODO |
| `MetricQuery` | namespace, metricName, dimensions, stat, unit | TODO |

### Frontend Pages

| Page | File | Status | Notes |
|------|------|--------|-------|
| NOC Dashboard | `src/pages/NocDashboard.tsx` | TODO | Main NOC dashboard with service grid |
| Automation | `src/pages/Automation.tsx` | TODO | CI/CD and automation dashboard |
| Dashboard | `src/pages/Dashboard.tsx` | TODO | General dashboard page |
| Home | `src/pages/Home.tsx` | TODO | Landing page |
| Settings | `src/pages/Settings.tsx` | TODO | App settings |

### Frontend Components

| Component | File | Status | Notes |
|-----------|------|--------|-------|
| OverviewPanel | `src/components/OverviewPanel.tsx` | TODO | Dashboard overview stats |
| AlertsPanel | `src/components/AlertsPanel.tsx` | TODO | Alert list with severity |
| CloudWatchPanel | `src/components/CloudWatchPanel.tsx` | TODO | CloudWatch metrics display |
| OperationsPanel | `src/components/OperationsPanel.tsx` | TODO | Events, actions, maintenance |
| ServiceCard | `src/components/ServiceCard.tsx` | Partial | Compact service health card |
| StatusBadge | `src/components/ServiceCard.tsx` | Partial | Status indicator badge |
| ServiceDetail | `src/components/ServiceDetail.tsx` | Partial | Full service detail modal |
| ResponseTimeChart | `src/components/ResponseTimeChart.tsx` | TODO | Response time visualization |
| AutomationDashboard | `src/components/AutomationDashboard.tsx` | TODO | DSKanban automation stats |

### Frontend Hooks

| Hook | File | Status | Notes |
|------|------|--------|-------|
| useDashboard | `src/hooks/useDashboard.ts` | TODO | Dashboard data polling (10s interval) |
| useAutomation | `src/hooks/useAutomation.ts` | TODO | DSKanban automation data (30s interval) |
| useAutomationQueue | `src/hooks/useAutomation.ts` | TODO | Queue status for next items |
| useConsent | `src/hooks/useConsent.ts` | TODO | Cookie consent management |
| useTenantTheme | `src/hooks/useTenantTheme.ts` | TODO | Tenant theming support |

### Frontend Types (`src/types.ts`)

| Type | Key Fields | Status |
|------|------------|--------|
| `ServiceHealth` | status, version, uptime, timestamp, service, environment, checks, memory, cpu, connections, responseTimeMs | Partial |
| `HealthCheck` | status, latencyMs, message | Partial |
| `MemoryStats` | heapUsedMB, heapTotalMB, rssMB, percentUsed | Partial |
| `CpuStats` | loadAverage, percentUsed | Partial |
| `ConnectionStats` | database, http | Partial |
| `ServiceConfig` | id, name, url, healthEndpoint, criticalService | TODO |
| `DashboardState` | services, lastUpdated, overallStatus | Partial |
| `HistoricalDataPoint` | timestamp, responseTimeMs, status, errorRate | Partial |
| `ServiceHistory` | serviceId, dataPoints | Partial |

### API Client (`src/api/`)

| File | Purpose | Status |
|------|---------|--------|
| `client.ts` | Base API client with fetch wrapper | TODO |
| `dskanban.ts` | DSKanban API integration for automation stats | TODO |
| `index.ts` | API exports | TODO |

---

## Functional Requirements

### FR-NOC-001: Service Health Monitoring Dashboard

Display real-time health status for all monitored DigiStratum services in a centralized dashboard.

**Acceptance Criteria:**
- Dashboard displays all registered services
- Each service shows current health status (healthy/degraded/unhealthy)
- Visual status indicators (color-coded badges)
- Compact card layout with expandable details

**@implements:**
- `frontend/src/components/ServiceCard.tsx::ServiceCard` - Compact service health display card
- `frontend/src/components/StatusBadge.tsx::StatusBadge` - Animated status indicator
- `frontend/src/types/service.ts::DashboardState` - Dashboard state type
- `backend/internal/health/health.go::HealthResponse` - Health response structure

---

### FR-NOC-002: Real-Time Service Status Updates

Provide automatic refresh of service status without page reload.

**Acceptance Criteria:**
- Dashboard auto-refreshes at configurable interval (default: 30s)
- Last updated timestamp displayed
- Manual refresh trigger available
- Graceful handling of refresh failures

**@implements:**
- `frontend/src/types/service.ts::DashboardState` - lastUpdated field
- `backend/internal/health/health.go::HealthResponse` - Timestamp field

**TODO:** Dashboard polling/websocket implementation pending page creation

---

### FR-NOC-003: CloudWatch Metrics Integration

Integrate with AWS CloudWatch for metrics visualization and alerting.

**Acceptance Criteria:**
- Pull key metrics from CloudWatch (latency, errors, invocations)
- Display metrics in service detail view
- Support for custom metric dashboards

**@implements:**
- `backend/internal/health/health.go::GetUptimePercent` - Stub for CloudWatch metrics integration

**TODO:** Full CloudWatch integration pending - see health.go line 232 for implementation notes

---

### FR-NOC-004: Alerts Panel with Severity Levels

Display active alerts organized by severity level (critical, warning, info).

**Acceptance Criteria:**
- Alerts panel visible on main dashboard
- Severity color coding (red/yellow/blue)
- Alert acknowledge/dismiss actions
- Alert history view

**TODO:** Alerts panel implementation pending

---

### FR-NOC-005: Operations Panel (Events, Actions, Maintenance)

Provide operations context including recent events, available actions, and maintenance windows.

**Acceptance Criteria:**
- Recent events timeline
- Quick action buttons (restart, scale, deploy)
- Maintenance window scheduler
- Event filtering by type/service

**TODO:** Operations panel implementation pending

---

### FR-NOC-006: Automation Dashboard (CI/CD Status, Runbooks)

Display CI/CD pipeline status and provide access to runbook documentation.

**Acceptance Criteria:**
- GitHub Actions workflow status for all DS repos
- Recent deployment history
- Runbook quick-links per service
- Pipeline trigger capability

**TODO:** Automation dashboard implementation pending

---

### FR-NOC-007: Service Detail Drill-Down View

Provide detailed service information in a modal/expanded view.

**Acceptance Criteria:**
- Full service metrics display
- Resource utilization (CPU, memory)
- Connection pool status
- Dependency health checks
- Historical data visualization

**@implements:**
- `frontend/src/components/ServiceDetail.tsx::ServiceDetail` - Full service health detail modal
- `frontend/src/types/service.ts::ServiceHealth` - Full health data structure
- `frontend/src/types/service.ts::MemoryStats` - Memory utilization type
- `frontend/src/types/service.ts::CpuStats` - CPU utilization type
- `frontend/src/types/service.ts::ConnectionStats` - Connection pool type
- `frontend/src/types/service.ts::HealthCheck` - Dependency check type

---

### FR-NOC-008: Response Time Charting

Display response time trends for monitored services.

**Acceptance Criteria:**
- Line chart showing response time over time
- Configurable time window (1h, 6h, 24h, 7d)
- Service comparison view
- Anomaly highlighting

**@implements:**
- `frontend/src/types/service.ts::HistoricalDataPoint` - Time series data point type
- `frontend/src/types/service.ts::ServiceHistory` - Service history container
- `backend/internal/health/health.go::LatencyMetrics` - Latency statistics

**TODO:** Chart visualization component pending

---

## Non-Functional Requirements

### NFR-NOC-001: Dashboard Refresh Interval < 30s

The dashboard must refresh service status within 30 seconds to maintain operational awareness.

**Acceptance Criteria:**
- Default refresh interval: 30 seconds
- Configurable via environment variable
- Visual indicator during refresh
- No data staleness > 30s under normal conditions

**@implements:**
- `frontend/src/types/service.ts::DashboardState` - State structure supports refresh tracking

**TODO:** Polling implementation with configurable interval pending

---

### NFR-NOC-002: Health Check Timeout 5s Per Service

Individual health checks must complete within 5 seconds to prevent cascade failures.

**Acceptance Criteria:**
- HTTP client timeout: 5000ms
- Graceful timeout handling
- Timeout logged as degraded/unhealthy

**@implements:**
- `backend/internal/health/health.go::DependencyConfig` - TimeoutMs field (default 5000)
- `backend/internal/health/health.go::CheckDependency` - Timeout enforcement with context
- `backend/internal/health/health.go::loadConfigFromEnv` - Default timeout configuration

---

### NFR-NOC-003: Concurrent Health Checks to All Monitored Services

Health checks must run in parallel to minimize total dashboard refresh time.

**Acceptance Criteria:**
- Parallel execution using goroutines
- WaitGroup synchronization
- Per-request context cancellation
- No blocking between service checks

**@implements:**
- `backend/internal/health/health.go::CheckDependenciesParallel` - Parallel health check implementation
- `backend/internal/health/health.go::CalculateOverallStatus` - Aggregate status from parallel results
- `backend/internal/health/health.go::CalculateLatencyMetrics` - Aggregate latency metrics

---

## Implementation Status Summary

| Requirement | Status | Coverage |
|-------------|--------|----------|
| FR-NOC-001 | Partial | Components exist, dashboard page pending |
| FR-NOC-002 | Partial | Types defined, polling pending |
| FR-NOC-003 | Stub | CloudWatch stub in health.go |
| FR-NOC-004 | TODO | Not implemented |
| FR-NOC-005 | TODO | Not implemented |
| FR-NOC-006 | TODO | Not implemented |
| FR-NOC-007 | Complete | ServiceDetail component fully implemented |
| FR-NOC-008 | Partial | Types defined, chart component pending |
| NFR-NOC-001 | Partial | Types support it, implementation pending |
| NFR-NOC-002 | Complete | Backend timeout handling implemented |
| NFR-NOC-003 | Complete | Parallel health checks implemented |

---

## Related Documents

- [DIGISTRATUM.md](../../.openclaw/workspace/DIGISTRATUM.md) - Ecosystem context
- [AGENTS.md](./AGENTS.md) - Development standards
- [PROJECT_CONTEXT.md](./PROJECT_CONTEXT.md) - App-specific context
