# E2E Test Directory

End-to-end tests for deployment validation and requirements traceability.

## Directory Structure

```
tests/e2e/
├── api/           # API endpoint tests (REST)
├── ui/            # Playwright UI tests
├── security/      # Auth, authz, rate-limit tests
├── fixtures/      # Shared test data and helpers
└── README.md      # This file
```

## Test Naming Conventions

### File Names

```
{resource}.{scenario}.test.ts
```

Examples:
- `items.crud.test.ts` — CRUD operations for items
- `items.validation.test.ts` — Input validation for items
- `items.tenant-isolation.test.ts` — Multi-tenant isolation tests

### Test Descriptions

Use descriptive `describe` and `it` blocks:

```typescript
describe('GET /api/items', () => {
  describe('authenticated user', () => {
    it('returns 200 with items for current tenant', () => { /* ... */ });
    it('returns empty array when no items exist', () => { /* ... */ });
  });

  describe('unauthenticated request', () => {
    it('returns 401 Unauthorized', () => { /* ... */ });
  });
});
```

## @covers Markers

Use `@covers` markers to link tests to requirements. This enables traceability from requirements → implementation → tests.

### Syntax

```typescript
/**
 * @covers FR-001 List items for current tenant
 * @covers FR-002 Filter items by status
 */
it('lists items filtered by status', async () => { /* ... */ });
```

### Marker Types

| Prefix | Meaning |
|--------|---------|
| FR-NNN | Functional Requirement |
| NFR-NNN | Non-Functional Requirement |
| SEC-NNN | Security Requirement |

### Multiple Markers

A single test can cover multiple requirements:

```typescript
/**
 * @covers FR-010 Create item with validation
 * @covers SEC-001 Input sanitization
 */
it('rejects items with XSS payloads in title', async () => { /* ... */ });
```

### Finding Coverage Gaps

```bash
# Extract all @covers markers
grep -rh "@covers" tests/e2e/ | sort | uniq

# Compare against REQUIREMENTS.md
# (automation script TBD)
```

## Test Categories

### API Tests (`api/`)

Full endpoint coverage:
- **Happy path**: Valid requests succeed
- **401**: Unauthenticated requests rejected
- **403**: Unauthorized requests rejected (wrong tenant, role)
- **404**: Non-existent resources
- **400**: Invalid input rejected
- **Tenant isolation**: Users can't access other tenants' data

### UI Tests (`ui/`)

Critical user journeys:
- Navigation flows
- Form submissions
- Error state handling
- Accessibility (keyboard, screen reader)

### Security Tests (`security/`)

Auth/authz edge cases:
- Session expiry handling
- Role escalation attempts
- Rate limiting behavior
- CSRF protection
- Token refresh flows

## Running Tests

```bash
# All E2E tests
pnpm run test:e2e

# API tests only
pnpm run test:e2e:api

# UI tests only
pnpm run test:e2e:ui

# Security tests only
pnpm run test:e2e:security
```

## Environment

Tests expect these environment variables:

| Variable | Description |
|----------|-------------|
| `API_BASE_URL` | Backend API URL (e.g., `http://localhost:3001`) |
| `APP_BASE_URL` | Frontend URL (e.g., `http://localhost:5173`) |
| `TEST_USER_EMAIL` | Test user email |
| `TEST_USER_PASSWORD` | Test user password |

See `fixtures/` for test data setup helpers.

## CI Integration

E2E tests run as deployment gates:

| Stage | When | Tests | On Failure |
|-------|------|-------|------------|
| Post-deploy (dev) | After dev deploy | Full suite | Block promotion |
| Post-deploy (prod) | After prod deploy | Smoke tests | Trigger rollback |

See: [docs/ci-testing.md](../../docs/ci-testing.md)
