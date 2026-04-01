# Requirements

> Machine-parseable requirements document with traceability markers.
> Update this file as your app evolves to maintain requirements → implementation → test traceability.

---

## Auto-Generation Behavior

This document contains **AUTO-GENERATED markers** that scaffold commands append to:

| Marker | Scaffold Command | Purpose |
|--------|------------------|---------|
| `<!-- AUTO-GENERATED: add-endpoint -->` | `ds-scaffold add-endpoint` | New API endpoints |
| `<!-- AUTO-GENERATED: add-model -->` | `ds-scaffold add-model` | New data models |
| `<!-- AUTO-GENERATED: add-page -->` | `ds-scaffold add-page` | New UI pages |
| `<!-- AUTO-GENERATED: add-service -->` | `ds-scaffold add-service` | New services |
| `<!-- AUTO-GENERATED: add-middleware -->` | `ds-scaffold add-middleware` | New middleware |
| `<!-- AUTO-GENERATED: add-env-var -->` | `ds-scaffold add-env-var` | New config vars |

**How it works:**
1. Scaffold commands find the marker comment
2. New requirements are appended below the marker
3. Each requirement gets an auto-incremented ID (e.g., `[FR-API-004]`)
4. Generated entries include timestamp and scaffold source

**Manual entries:** You can add requirements manually above or below the auto-generated section. The scaffold only appends after the marker.

---

## Marker Format

Requirements use the format `[FR-{CATEGORY}-{NNN}]` for machine parsing:

- **FR** = Functional Requirement (or **NFR** for Non-Functional)
- **CATEGORY** = Domain grouping (API, UI, SEC, AUTH, DATA, SVC, MW, CFG, etc.)
- **NNN** = Zero-padded sequence number within category

**Examples:**
- `[FR-API-001]` — First API requirement
- `[FR-UI-003]` — Third UI requirement
- `[NFR-PERF-002]` — Second performance requirement

### Adding New Requirements

1. Choose the appropriate category (or create a new one)
2. Find the next available sequence number in that category
3. Add the requirement with marker, description, and acceptance criteria
4. Update code to reference the marker (see Traceability below)

---

## Traceability Conventions

Link requirements to implementation and tests using annotations:

### In Code (`@implements`)

```go
// @implements FR-API-001
func HealthHandler(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

```typescript
// @implements FR-UI-001
export function Dashboard() {
    return <DashboardView />;
}
```

### In Tests (`@covers`)

```go
// TestHealthEndpoint @covers FR-API-001
func TestHealthEndpoint(t *testing.T) {
    // Test implementation
}
```

```typescript
// @covers FR-UI-001
test('dashboard displays user data', async () => {
    // Test implementation
});
```

### Traceability Matrix

Run the requirements scanner to generate coverage:

```bash
go run ./tools/cmd/req-scanner --root . --output coverage.json
```

This extracts all `[FR-*]` markers and maps them to `@implements`/`@covers` annotations.

---

## Functional Requirements

### API Endpoints

<!-- AUTO-GENERATED: add-endpoint -->
- **[FR-API-001]** `GET /api/health` — Health check endpoint
  - Returns 200 with `{"status": "ok"}` when service is healthy
  - Returns 503 with error details when unhealthy
  - No authentication required

- **[FR-API-002]** `GET /api/me` — Current user info
  - Returns authenticated user profile
  - Returns 401 if not authenticated
  - Response includes user ID, email, tenant context

- **[FR-API-003]** `GET /api` — API discovery endpoint
  - Returns HAL-format links to all available resources
  - Unauthenticated: shows public endpoints only
  - Authenticated: shows permitted endpoints for user

### Data Models

<!-- AUTO-GENERATED: add-model -->
(No models scaffolded yet)

### Pages

<!-- AUTO-GENERATED: add-page -->
- **[FR-UI-001]** `/dashboard` — User dashboard (protected)
  - Displays after successful authentication
  - Shows tenant-scoped summary data
  - Accessible via main navigation

- **[FR-UI-002]** `/settings` — User settings (protected)
  - Allows preference configuration
  - Changes persist across sessions
  - Validates input before saving

- **[FR-UI-003]** `/` — Landing/home page
  - Redirects authenticated users to dashboard
  - Shows login prompt for unauthenticated users

### Services

<!-- AUTO-GENERATED: add-service -->
(No services scaffolded yet)

---

## Non-Functional Requirements

### Security Requirements

- **[FR-SEC-001]** All `/api/*` endpoints require authentication
  - Exception: `/api/health`, `/api/build` (public)
  - Returns 401 for missing/invalid session
  - Session validated via DSAccount SSO

- **[FR-SEC-002]** Session expiry handled gracefully
  - Frontend detects 401 responses
  - User redirected to login with return URL
  - In-progress work preserved where possible

- **[FR-SEC-003]** Tenant isolation enforced on all data access
  - Cross-tenant requests return 403
  - All queries include tenant context
  - Audit log captures tenant in all operations

### Data Requirements

- **[FR-DATA-001]** All mutations are idempotent
  - Retry-safe for network failures
  - Uses optimistic locking where applicable

- **[FR-DATA-002]** Soft delete for user data
  - Deleted records marked, not removed
  - Retention policy configurable
  - Hard delete available for compliance

### Middleware

<!-- AUTO-GENERATED: add-middleware -->
(No middleware scaffolded yet)

### Configuration

<!-- AUTO-GENERATED: add-env-var -->
(No config scaffolded yet)

---

## Detailed Requirements

For comprehensive requirements with acceptance criteria and verification methods, see:

### Functional Requirements
- [FR-AUTH](requirements/fr-auth.md) — Authentication & Authorization
- [FR-TENANT](requirements/fr-tenant.md) — Multi-Tenant Support
- [FR-NAV](requirements/fr-nav.md) — Navigation
- [FR-THEME](requirements/fr-theme.md) — Theming
- [FR-I18N](requirements/fr-i18n.md) — Internationalization
- [FR-API](requirements/fr-api.md) — API Standards

### Non-Functional Requirements
- [NFR-PERF](requirements/nfr-performance.md) — Performance
- [NFR-SEC](requirements/nfr-security.md) — Security
- [NFR-A11Y](requirements/nfr-accessibility.md) — Accessibility
- [NFR-OBS](requirements/nfr-observability.md) — Observability
- [NFR-TEST](requirements/nfr-testing.md) — Testing

---

## Maintenance

### When to Update This Document

- **New feature:** Add requirement markers before implementation
- **Bug fix:** Reference existing requirement or add new one if gap found
- **Refactor:** Ensure `@implements` annotations move with code
- **Test addition:** Add `@covers` annotations linking to requirements

### Validation

```bash
# Check all requirements have implementations
go run ./tools/cmd/req-scanner --check-coverage

# List orphaned implementations (no requirement)
go run ./tools/cmd/req-scanner --find-orphans

# Generate HTML traceability report
go run ./tools/cmd/req-scanner --html report.html
```
