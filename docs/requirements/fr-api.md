# FR-API: API Standards

> Standard API conventions for DS ecosystem applications.
> RESTful design with HAL hypermedia and consistent error handling.

---

## Requirements

### FR-API-001: All API responses follow HAL format

API responses use HAL (Hypertext Application Language) for hypermedia-driven navigation.

**Acceptance Criteria:**
1. Successful responses include `_links` object with navigable relations
2. Collection responses include `_embedded` for nested resources
3. Self-link (`_links.self`) present on all resources
4. Content-Type: `application/hal+json` for HAL responses
5. Pagination links included for collection endpoints: `first`, `prev`, `next`, `last`
6. Error responses use standard JSON (not HAL)

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/api/hal_test.go` |
| Integration test | `backend/test/integration/api_test.go:TestHALFormat` |
| Contract test | OpenAPI spec validation |

**Evidence:**
- CI test results
- Example response validation against HAL spec

**Example Response:**
```json
{
  "_links": {
    "self": { "href": "/api/items/123" },
    "collection": { "href": "/api/items" },
    "tenant": { "href": "/api/tenants/456" }
  },
  "id": "123",
  "name": "Example Item",
  "created_at": "2026-03-23T10:00:00Z"
}
```

---

### FR-API-002: Discovery endpoint lists all available operations

Root API endpoint provides hypermedia discovery of available resources.

**Acceptance Criteria:**
1. `GET /api` returns links to all top-level resources
2. Links include `rel` names describing the relationship
3. Links indicate allowed HTTP methods (if applicable)
4. Response includes API version information
5. Unauthenticated requests show only public endpoints
6. Authenticated requests show all permitted endpoints

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/api/discovery_test.go` |
| E2E test | `frontend/e2e/api-integration.spec.ts:discovery` |

**Evidence:**
- CI test results
- `curl https://api.example.com/api` shows discovery document

**Example Response:**
```json
{
  "_links": {
    "self": { "href": "/api" },
    "health": { "href": "/api/health" },
    "build": { "href": "/api/build" },
    "items": { "href": "/api/items", "methods": ["GET", "POST"] },
    "auth": { "href": "/api/auth/me" }
  },
  "version": "1.0.0",
  "app": "ds-app-template"
}
```

---

### FR-API-003: Health endpoint available at /api/health

Health check endpoint for load balancer and monitoring integration.

**Acceptance Criteria:**
1. `GET /api/health` returns 200 OK when service is healthy
2. Response includes `status` field: "healthy", "degraded", or "unhealthy"
3. Shallow check (no auth): verifies Lambda is running
4. Deep check (`?deep=true`, authenticated): checks all dependencies
5. Deep check includes latency metrics for each dependency
6. Unhealthy status returns 503 Service Unavailable
7. No authentication required for shallow health check

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/health/handler_test.go` |
| Integration test | `backend/test/integration/health_test.go` |
| E2E test | `frontend/e2e/api-integration.spec.ts:health check` |

**Evidence:**
- CI test results
- Load balancer health check logs
- CloudWatch alarm on health endpoint failures

**Shallow Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-03-23T10:00:00Z"
}
```

**Deep Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-03-23T10:00:00Z",
  "dependencies": {
    "dynamodb": { "status": "healthy", "latency_ms": 12 },
    "dsaccount": { "status": "healthy", "latency_ms": 45 }
  },
  "version": "1.2.3",
  "environment": "prod"
}
```

---

### FR-API-004: Build info available at /api/build

Build metadata endpoint for deployment verification and debugging.

**Acceptance Criteria:**
1. `GET /api/build` returns build information
2. Includes: version, git commit hash, build timestamp
3. Includes: environment name (dev/staging/prod)
4. No sensitive information exposed (no secrets, internal paths)
5. No authentication required
6. Cache headers allow CDN caching (immutable per deployment)

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/api/build_test.go` |
| E2E test | `frontend/e2e/api-integration.spec.ts:build info` |

**Evidence:**
- CI test results
- `curl https://api.example.com/api/build` returns expected format

**Example Response:**
```json
{
  "version": "1.2.3",
  "commit": "abc123def",
  "build_time": "2026-03-23T09:00:00Z",
  "environment": "prod",
  "app": "ds-app-template"
}
```

---

## Implementation

### HAL Response Builder

```go
// backend/internal/api/hal.go
type HALResource struct {
    Links    map[string]HALLink `json:"_links"`
    Embedded map[string]any     `json:"_embedded,omitempty"`
}

type HALLink struct {
    Href    string   `json:"href"`
    Methods []string `json:"methods,omitempty"`
}

func NewHALResource(selfHref string) *HALResource {
    return &HALResource{
        Links: map[string]HALLink{
            "self": {Href: selfHref},
        },
    }
}

func (r *HALResource) AddLink(rel, href string) *HALResource {
    r.Links[rel] = HALLink{Href: href}
    return r
}

func (r *HALResource) WithMethods(rel string, methods ...string) *HALResource {
    if link, ok := r.Links[rel]; ok {
        link.Methods = methods
        r.Links[rel] = link
    }
    return r
}
```

### Error Response Format

```go
// Standard error response (not HAL)
type ErrorResponse struct {
    Error   string            `json:"error"`
    Message string            `json:"message,omitempty"`
    Fields  map[string]string `json:"fields,omitempty"`
    TraceID string            `json:"trace_id,omitempty"`
}

func WriteError(w http.ResponseWriter, status int, err ErrorResponse) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(err)
}
```

### Health Handler

```go
// backend/internal/health/handler.go
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
    deep := r.URL.Query().Get("deep") == "true"
    
    if deep {
        // Deep check requires auth
        if auth.GetSession(r.Context()) == nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        h.deepHealth(w, r)
        return
    }
    
    // Shallow check
    json.NewEncoder(w).Encode(map[string]any{
        "status":    "healthy",
        "timestamp": time.Now().UTC(),
    })
}
```

---

## Error Codes

| HTTP Status | Usage |
|-------------|-------|
| 200 OK | Successful GET, PUT |
| 201 Created | Successful POST (resource created) |
| 204 No Content | Successful DELETE |
| 400 Bad Request | Validation errors, malformed request |
| 401 Unauthorized | Missing or invalid authentication |
| 403 Forbidden | Authenticated but not permitted |
| 404 Not Found | Resource doesn't exist |
| 409 Conflict | Resource state conflict |
| 422 Unprocessable | Validation passed but business rule failed |
| 429 Too Many Requests | Rate limited |
| 500 Internal Server Error | Unexpected server error |
| 503 Service Unavailable | Dependency failure, maintenance |

---

## Traceability

| Requirement | Implementation | Test | Status |
|-------------|----------------|------|--------|
| FR-API-001 | `backend/internal/api/hal.go` | `hal_test.go`, `api_test.go` | ⚠️ |
| FR-API-002 | `backend/internal/api/discovery.go` | `discovery_test.go`, `api-integration.spec.ts` | ⚠️ |
| FR-API-003 | `backend/internal/health/handler.go` | `handler_test.go`, `health_test.go` | ⚠️ |
| FR-API-004 | `backend/internal/api/build.go` | `build_test.go`, `api-integration.spec.ts` | ⚠️ |

---

*Last updated: 2026-03-23*
