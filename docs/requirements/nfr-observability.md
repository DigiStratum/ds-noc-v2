# NFR-OBS: Observability Requirements

> Monitoring, logging, and alerting standards for DigiStratum applications.
> Implements structured logging, metrics dashboards, and correlation ID propagation.

---

## Audit-Ready Summary

### NFR-OBS-001: Structured JSON Logging

**Acceptance Criteria:**
1. All logs output as JSON (not plain text)
2. Every log entry includes: `timestamp`, `level`, `message`, `correlation_id`
3. Request logs include: `method`, `path`, `status`, `duration_ms`
4. Error logs include: `error` field with message (no stack traces in prod)
5. No sensitive data logged (tokens, passwords, PII)
6. Logs queryable via CloudWatch Logs Insights

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/middleware/logging_test.go` |
| Manual check | CloudWatch Logs → sample log entries |

**Evidence:** CloudWatch Logs Insights query returns structured JSON

---

### NFR-OBS-002: Error Rate Alerting < 1%

**Acceptance Criteria:**
1. CloudWatch alarm configured for error rate > 5% (2/3 periods)
2. 5XX error alarm triggers if > 10 errors/minute
3. Alarms send notifications via SNS
4. Staging and prod environments have alarms enabled
5. Dev environment has alarms disabled (no noise)

**Verification:**
| Method | Location |
|--------|----------|
| CDK config | `infra/lib/monitoring.ts` |
| AWS Console | CloudWatch → Alarms |

**Evidence:** CloudWatch alarm configuration, no alarm triggers (healthy state)

---

### NFR-OBS-003: Latency Percentile Dashboards

**Acceptance Criteria:**
1. CloudWatch dashboard exists for each environment
2. Dashboard shows: p50, p95, p99 latency for API Gateway
3. Dashboard shows: Lambda duration, DynamoDB latency
4. p95 target line (500ms) shown as annotation
5. Dashboard auto-refreshes (1-minute interval)

**Verification:**
| Method | Location |
|--------|----------|
| CDK config | `infra/lib/monitoring.ts` |
| AWS Console | CloudWatch → Dashboards |

**Evidence:** Dashboard screenshot showing latency graphs

---

### NFR-OBS-004: Request Correlation ID Propagation

**Acceptance Criteria:**
1. Every request assigned `X-Correlation-ID` header (generated if not present)
2. Correlation ID included in all log entries for that request
3. Correlation ID returned in response header
4. Cross-service calls propagate correlation ID
5. Can trace full request flow via correlation ID in Logs Insights

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/middleware/correlation_test.go` |
| E2E test | `frontend/e2e/api-integration.spec.ts:correlation ID` |
| Manual test | `curl -v` shows X-Correlation-ID in response |

**Evidence:** Logs Insights query by correlation_id returns complete request flow

---

### NFR-OBS-005: Health Check Endpoint

**Acceptance Criteria:**
1. `GET /api/health` returns 200 when healthy
2. Shallow check (no auth) returns `{"status":"healthy","timestamp":"..."}`
3. Deep check (`?deep=true`, authenticated) checks all dependencies
4. Deep check returns latency_ms for each dependency
5. Unhealthy dependencies return 503 with `status:"degraded"`
6. Health endpoint used by load balancer health checks

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/health/handler_test.go` |
| Integration test | `backend/test/integration/health_test.go` |
| Manual test | `curl https://app.digistratum.com/api/health` |

**Evidence:** CI test results, load balancer health check configuration

---

## Quick Reference

| Requirement | Description | Target |
|-------------|-------------|--------|
| NFR-OBS-001 | Structured logging to CloudWatch | JSON format |
| NFR-OBS-002 | Error rate alerting | < 1% threshold |
| NFR-OBS-003 | Latency percentile dashboards | p95, p99 visibility |
| NFR-OBS-004 | Request correlation ID propagation | End-to-end tracing |
| NFR-OBS-005 | Health check endpoint | /api/health |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          OBSERVABILITY FLOW                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  CloudFront ──┐                                                             │
│               │                                                             │
│  API Gateway ─┼──► CloudWatch Metrics ──► CloudWatch Alarms                 │
│               │           │                      │                          │
│  Lambda ──────┤           │                      │                          │
│               │           ▼                      ▼                          │
│  DynamoDB ────┘    CloudWatch Dashboard    SNS Topic ──► PagerDuty         │
│                                                  │                          │
│  Lambda Logs ────► CloudWatch Logs              │──► Email                 │
│                          │                       │                          │
│                          ▼                       │──► Slack                 │
│                   Logs Insights                  │                          │
│                   (Queries)                                                 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## NFR-OBS-001: Structured Logging

**Target:** All logs in JSON format for machine parsing and CloudWatch Logs Insights queries.

### Log Format

```json
{
  "timestamp": "2026-03-22T10:30:45.123Z",
  "level": "INFO",
  "message": "Request completed",
  "correlation_id": "req-abc123",
  "tenant_id": "tenant-xyz",
  "user_id": "user-123",
  "method": "GET",
  "path": "/api/items",
  "status": 200,
  "duration_ms": 45,
  "request_id": "lambda-req-456"
}
```

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | ISO 8601 | When the event occurred |
| `level` | string | DEBUG, INFO, WARN, ERROR |
| `message` | string | Human-readable description |
| `correlation_id` | string | Request trace identifier |
| `request_id` | string | AWS request ID |

### Contextual Fields

| Field | When Present | Description |
|-------|--------------|-------------|
| `tenant_id` | Authenticated requests | Multi-tenant isolation |
| `user_id` | Authenticated requests | User audit trail |
| `method` | HTTP requests | HTTP method |
| `path` | HTTP requests | Request path |
| `status` | HTTP responses | Response status code |
| `duration_ms` | Request completion | Processing time |
| `error` | Error events | Error details |
| `stack_trace` | Error events | Stack trace (non-prod) |

### Go Implementation

```go
package middleware

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "time"
)

func init() {
    // JSON handler for structured logging
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)
}

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Wrap response writer to capture status
        wrapped := &responseWriter{ResponseWriter: w, status: 200}
        
        // Get correlation ID from context
        correlationID := r.Context().Value(CorrelationIDKey).(string)
        
        // Process request
        next.ServeHTTP(wrapped, r)
        
        // Log completion
        slog.Info("request_completed",
            "correlation_id", correlationID,
            "method", r.Method,
            "path", r.URL.Path,
            "status", wrapped.status,
            "duration_ms", time.Since(start).Milliseconds(),
            "user_agent", r.UserAgent(),
            "remote_addr", r.RemoteAddr,
        )
    })
}

type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
}
```

### Log Levels

| Level | Usage | Example |
|-------|-------|---------|
| DEBUG | Development details | Query parameters, intermediate values |
| INFO | Normal operations | Request completed, user logged in |
| WARN | Unusual but handled | Retry succeeded, rate limit approached |
| ERROR | Failures requiring attention | Database error, auth failure |

### Security Logging

```go
// DO log security events
slog.Info("auth_event",
    "event", "login_success",
    "user_id", session.UserID,
    "correlation_id", correlationID,
)

slog.Warn("auth_event",
    "event", "login_failure",
    "reason", "invalid_token",
    "ip", getClientIP(r),
    "correlation_id", correlationID,
)

// NEVER log sensitive data
// ❌ token, password, session data, PII
```

### CloudWatch Logs Insights Queries

```sql
-- Find errors by correlation ID
fields @timestamp, @message
| filter correlation_id = "req-abc123"
| sort @timestamp desc

-- Error rate by path
fields @timestamp, path, level
| filter level = "ERROR"
| stats count(*) as errors by path
| sort errors desc

-- Slow requests (p95)
fields @timestamp, path, duration_ms
| filter duration_ms > 0
| stats percentile(duration_ms, 95) as p95 by path
| sort p95 desc

-- User activity audit
fields @timestamp, @message, user_id, path, method
| filter user_id = "user-123"
| sort @timestamp desc
| limit 100
```

---

## NFR-OBS-002: Error Rate Alerting

**Target:** Alert when error rate exceeds 1% of requests.

### Alarm Configuration

| Alarm | Metric | Threshold | Evaluation |
|-------|--------|-----------|------------|
| High Error Rate | Lambda error rate | > 5% | 2/3 periods (5m) |
| 5XX Errors | API Gateway 5XX | > 10/min | 2/2 periods (1m) |
| Lambda Errors | Lambda Errors | > 5/min | 2/2 periods (1m) |
| Lambda Throttles | Lambda Throttles | > 0 | 1/1 periods (1m) |
| DynamoDB Throttles | ThrottledRequests | > 0 | 1/1 periods (1m) |

### CDK Implementation

```typescript
// cdk/lib/constructs/monitoring.ts

const errorRateAlarm = new cloudwatch.Alarm(this, 'ErrorRateAlarm', {
  alarmName: `${appName}-${environment}-high-error-rate`,
  alarmDescription: `Error rate exceeded 5% for ${appName} in ${environment}`,
  metric: new cloudwatch.MathExpression({
    expression: '(errors / invocations) * 100',
    usingMetrics: {
      errors: lambdaFunction.metricErrors(),
      invocations: lambdaFunction.metricInvocations(),
    },
  }),
  threshold: 5,
  evaluationPeriods: 3,
  datapointsToAlarm: 2,
  treatMissingData: cloudwatch.TreatMissingData.NOT_BREACHING,
});

errorRateAlarm.addAlarmAction(new actions.SnsAction(alertTopic));
errorRateAlarm.addOkAction(new actions.SnsAction(alertTopic));
```

### Environment-Specific Behavior

| Environment | Alarms Enabled | Notifications |
|-------------|----------------|---------------|
| dev | No | No |
| staging | Yes | Yes |
| prod | Yes | Yes |

### SNS Integration

```typescript
const alertTopic = new sns.Topic(this, 'AlertTopic', {
  topicName: `${appName}-alerts-${environment}`,
  displayName: `${appName} ${environment} Alerts`,
});

// Email subscription
alertTopic.addSubscription(
  new subscriptions.EmailSubscription('ops@digistratum.com')
);

// PagerDuty integration (production)
if (environment === 'prod') {
  alertTopic.addSubscription(
    new subscriptions.UrlSubscription(pagerDutyIntegrationUrl)
  );
}
```

---

## NFR-OBS-003: Latency Dashboards

**Target:** Dashboard with p50, p95, p99 latency metrics for all services.

### Performance Baselines

```typescript
export const PerformanceBaselines = {
  /** Target API response time p95 (ms) */
  apiLatencyP95Ms: 500,
  
  /** Target API response time p99 (ms) */
  apiLatencyP99Ms: 1000,
  
  /** Maximum acceptable error rate (%) */
  maxErrorRatePercent: 1,
  
  /** Target availability (%) */
  availabilityTarget: 99.9,
  
  /** Lambda cold start threshold (ms) */
  coldStartThresholdMs: 1000,
  
  /** DynamoDB read latency target (ms) */
  dynamoReadLatencyMs: 25,
  
  /** DynamoDB write latency target (ms) */
  dynamoWriteLatencyMs: 50,
};
```

### Dashboard Layout

**Row 1: Key Metrics (Single Value Widgets)**
- Requests (5m)
- Error Rate %
- P95 Latency (ms)
- 5XX Errors
- Concurrent Executions

**Row 2: Request Volume & Errors**
- API Requests over time
- 4XX vs 5XX error trends

**Row 3: Latency**
- API Gateway latency (p50, p95, p99) with target annotations
- Lambda duration distribution

**Row 4: DynamoDB**
- Read/Write latency
- Capacity utilization and throttles

### CDK Dashboard Implementation

```typescript
const dashboard = new cloudwatch.Dashboard(this, 'Dashboard', {
  dashboardName: `${appName}-${environment}`,
});

// Latency graph with targets
dashboard.addWidgets(
  new cloudwatch.GraphWidget({
    title: 'API Latency',
    left: [
      new cloudwatch.Metric({
        namespace: 'AWS/ApiGateway',
        metricName: 'Latency',
        dimensionsMap: { ApiId: apiId },
        statistic: 'p50',
        label: 'p50',
      }),
      new cloudwatch.Metric({
        namespace: 'AWS/ApiGateway',
        metricName: 'Latency',
        dimensionsMap: { ApiId: apiId },
        statistic: 'p95',
        label: 'p95',
      }),
      new cloudwatch.Metric({
        namespace: 'AWS/ApiGateway',
        metricName: 'Latency',
        dimensionsMap: { ApiId: apiId },
        statistic: 'p99',
        label: 'p99',
      }),
    ],
    leftAnnotations: [
      { value: 500, label: 'p95 Target', color: '#ff7f0e' },
      { value: 1000, label: 'p99 Target', color: '#d62728' },
    ],
    width: 12,
    height: 6,
  })
);
```

### Key Metrics Monitored

| Service | Metric | Purpose |
|---------|--------|---------|
| Lambda | Invocations | Request volume |
| Lambda | Errors | Application failures |
| Lambda | Duration (p95/p99) | Performance |
| Lambda | Throttles | Capacity issues |
| Lambda | ConcurrentExecutions | Scalability |
| API Gateway | Count | Request volume |
| API Gateway | 4XXError | Client errors |
| API Gateway | 5XXError | Server errors |
| API Gateway | Latency (p95/p99) | Performance |
| DynamoDB | ConsumedRCU/WCU | Database load |
| DynamoDB | ThrottledRequests | Capacity issues |
| DynamoDB | SuccessfulRequestLatency | Database performance |

---

## NFR-OBS-004: Correlation ID Propagation

**Target:** End-to-end request tracing via correlation IDs.

### ID Generation

```go
package middleware

import (
    "context"
    "net/http"
    
    "github.com/google/uuid"
)

type contextKey string
const CorrelationIDKey contextKey = "correlation_id"

func CorrelationIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get from header or generate new
        correlationID := r.Header.Get("X-Correlation-ID")
        if correlationID == "" {
            correlationID = uuid.New().String()
        }
        
        // Set in response header
        w.Header().Set("X-Correlation-ID", correlationID)
        
        // Add to context
        ctx := context.WithValue(r.Context(), CorrelationIDKey, correlationID)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func GetCorrelationID(ctx context.Context) string {
    if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
        return id
    }
    return ""
}
```

### Header Propagation

| Header | Direction | Usage |
|--------|-----------|-------|
| `X-Correlation-ID` | Request & Response | Primary trace ID |
| `X-Request-ID` | Response | AWS Lambda request ID |

### Frontend Integration

```typescript
// api/client.ts
const apiClient = {
  async request(url: string, options: RequestInit = {}) {
    const correlationId = crypto.randomUUID();
    
    const response = await fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        'X-Correlation-ID': correlationId,
      },
    });
    
    // Log correlation ID for debugging
    console.debug('Request', {
      correlationId,
      url,
      status: response.status,
    });
    
    return response;
  }
};
```

### Cross-Service Propagation

```go
// When calling another service, propagate correlation ID
func (c *Client) CallExternalService(ctx context.Context, data interface{}) error {
    correlationID := GetCorrelationID(ctx)
    
    req, _ := http.NewRequestWithContext(ctx, "POST", serviceURL, body)
    req.Header.Set("X-Correlation-ID", correlationID)
    
    return c.httpClient.Do(req)
}
```

### Tracing Queries

```sql
-- Follow request through entire flow
fields @timestamp, @message, level, path, duration_ms
| filter correlation_id = "550e8400-e29b-41d4-a716-446655440000"
| sort @timestamp asc

-- Find all errors for a session
fields @timestamp, @message, error
| filter correlation_id LIKE "sess-abc%"
| filter level = "ERROR"
| sort @timestamp desc
```

---

## Health Check Endpoint

**Endpoint:** `GET /health`

### Shallow Health (Load Balancer)

```json
{
  "status": "healthy",
  "timestamp": "2026-03-22T10:30:45.123Z"
}
```

### Deep Health (Authenticated)

```json
{
  "status": "healthy",
  "timestamp": "2026-03-22T10:30:45.123Z",
  "dependencies": {
    "dynamodb": {
      "status": "healthy",
      "latency_ms": 12
    },
    "dsaccount": {
      "status": "healthy",
      "latency_ms": 45
    }
  },
  "version": "1.2.3",
  "environment": "prod"
}
```

### Implementation

```go
func (h *HealthHandler) DeepHealth(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    status := "healthy"
    
    deps := map[string]DependencyHealth{}
    
    // Check DynamoDB
    start := time.Now()
    if err := h.db.Ping(ctx); err != nil {
        deps["dynamodb"] = DependencyHealth{
            Status:  "unhealthy",
            Error:   err.Error(),
        }
        status = "degraded"
    } else {
        deps["dynamodb"] = DependencyHealth{
            Status:    "healthy",
            LatencyMs: time.Since(start).Milliseconds(),
        }
    }
    
    response := HealthResponse{
        Status:       status,
        Timestamp:    time.Now().UTC(),
        Dependencies: deps,
        Version:      os.Getenv("APP_VERSION"),
        Environment:  os.Getenv("ENVIRONMENT"),
    }
    
    statusCode := http.StatusOK
    if status != "healthy" {
        statusCode = http.StatusServiceUnavailable
    }
    
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(response)
}
```

---

## Runbooks

### High Error Rate

**Trigger:** Error rate > 5% for 10 minutes

**Steps:**
1. Check CloudWatch Logs Insights for error patterns
   ```sql
   fields @timestamp, @message, error, path
   | filter level = "ERROR"
   | sort @timestamp desc
   | limit 50
   ```
2. Identify affected endpoints
3. Check recent deployments
4. If deployment-related, rollback
5. If dependency failure, check DynamoDB/external services

### High Latency

**Trigger:** p95 latency > 500ms for 5 minutes

**Steps:**
1. Check Lambda duration metrics
2. Check DynamoDB consumed capacity
3. Look for slow queries in logs
   ```sql
   fields @timestamp, path, duration_ms
   | filter duration_ms > 500
   | sort duration_ms desc
   | limit 20
   ```
4. If DynamoDB throttling, increase capacity
5. If cold starts, consider provisioned concurrency

### DynamoDB Throttling

**Trigger:** ThrottledRequests > 0

**Steps:**
1. Check consumed vs provisioned capacity
2. Identify hot partitions
3. Short-term: Increase capacity
4. Long-term: Review access patterns

---

## Cost Optimization

### Log Retention

| Environment | Retention |
|-------------|-----------|
| dev | 7 days |
| staging | 14 days |
| prod | 90 days |

### Metric Resolution

| Metric Type | Resolution |
|-------------|------------|
| Standard metrics | 1 minute |
| Custom metrics | 1 minute |
| Dashboard refresh | 1 minute |

### Log Level by Environment

| Environment | Default Level |
|-------------|---------------|
| dev | DEBUG |
| staging | INFO |
| prod | INFO |

---

## Traceability

| Requirement | Implementation | Verification |
|-------------|----------------|--------------|
| NFR-OBS-001 | slog JSON handler | Log format tests |
| NFR-OBS-002 | CloudWatch Alarms | Alarm trigger tests |
| NFR-OBS-003 | CloudWatch Dashboard | Visual inspection |
| NFR-OBS-004 | Correlation middleware | E2E trace tests |

---

*Last updated: 2026-03-22*
