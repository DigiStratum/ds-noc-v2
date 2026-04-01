# REQUIREMENTS.md - DS NOC v2

Requirements traceability document for ds-noc-v2 Network Operations Center application.

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
