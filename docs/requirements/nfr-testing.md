# NFR-TEST: Testing Requirements

> Testing standards for DigiStratum applications.
> Measurable coverage targets, enforcement policies, and exception processes.

---

## Audit-Ready Summary

### NFR-TEST-001: Backend Unit Test Coverage > 80%

**Acceptance Criteria:**
1. Overall backend coverage ≥ 80% as reported by `go test -cover`
2. `internal/auth` package coverage ≥ 85%
3. `internal/api` package coverage ≥ 80%
4. Coverage report generated on every CI run
5. CI fails if coverage drops below threshold

**Verification:**
| Method | Location |
|--------|----------|
| CI gate | `.github/workflows/ci.yml` |
| Script | `go test -coverprofile=coverage.out ./...` |

**Evidence:** CI workflow logs showing coverage percentage

---

### NFR-TEST-002: Frontend Unit Test Coverage ≥ 70%

**Acceptance Criteria:**
1. Statement coverage ≥ 70% (target, currently phased from 8%)
2. Branch coverage ≥ 60%
3. Function coverage ≥ 60%
4. Coverage thresholds enforced in `vite.config.ts`
5. CI fails if coverage drops below configured threshold

**Verification:**
| Method | Location |
|--------|----------|
| CI gate | `.github/workflows/ci.yml` |
| Config | `frontend/vite.config.ts` (coverage thresholds) |

**Evidence:** CI workflow logs, coverage report in `frontend/coverage/`

---

### NFR-TEST-003: Integration Tests for All API Endpoints

**Acceptance Criteria:**
1. Every API endpoint has at least one integration test
2. Integration tests use DynamoDB Local (real DB, not mocks)
3. Tests verify request/response contracts
4. Tests verify authentication enforcement
5. Tests verify tenant isolation

**Verification:**
| Method | Location |
|--------|----------|
| Integration tests | `backend/test/integration/` |
| CI job | `.github/workflows/ci.yml` (integration test job) |

**Evidence:** CI workflow logs showing integration test pass

---

### NFR-TEST-004: E2E Tests for Critical User Flows

**Acceptance Criteria:**
1. Authentication flow covered (`auth.spec.ts`)
2. Navigation flow covered (`navigation.spec.ts`)
3. Theme/i18n switching covered (`theme-i18n.spec.ts`)
4. Accessibility tests covered (`accessibility.spec.ts`)
5. All E2E tests pass in CI before deployment

**Verification:**
| Method | Location |
|--------|----------|
| E2E tests | `frontend/e2e/*.spec.ts` |
| CI job | `.github/workflows/ci.yml` (Playwright job) |

**Evidence:** Playwright test report in CI artifacts

---

### NFR-TEST-005: All Tests Must Pass for Deployment

**Acceptance Criteria:**
1. CI pipeline gates deployment on test passage
2. Branch protection requires status checks to pass
3. No manual bypass of test failures
4. Failed tests block merge to main
5. Deployment workflow depends on test workflow success

**Verification:**
| Method | Location |
|--------|----------|
| Branch protection | GitHub repo settings |
| CI config | `.github/workflows/ci.yml`, `.github/workflows/deploy.yml` |

**Evidence:** GitHub Actions workflow dependency graph

---

## Quick Reference

| Requirement | Target | Enforcement |
|-------------|--------|-------------|
| NFR-TEST-001 | Backend unit test coverage > 80% | CI gate |
| NFR-TEST-002 | Frontend unit test coverage ≥ 70% (phased) | CI gate |
| NFR-TEST-003 | Integration tests for all API endpoints | CI gate |
| NFR-TEST-004 | E2E tests for critical user flows | CI gate |
| NFR-TEST-005 | All tests must pass for deployment | CI gate |

---

## Test Pyramid Strategy

```
                    ┌─────────┐
                    │   E2E   │  ~10% of tests
                    │  Tests  │  Slow, expensive, high confidence
                    ├─────────┤
                    │ Integra-│  ~20% of tests
                    │  tion   │  Medium speed, real dependencies
                    ├─────────┤
                    │         │
                    │  Unit   │  ~70% of tests
                    │  Tests  │  Fast, isolated, granular
                    │         │
                    └─────────┘
```

### Layer Responsibilities

| Layer | Scope | Speed | Dependencies |
|-------|-------|-------|--------------|
| Unit | Single function/component | < 100ms | Mocked |
| Integration | Multiple components, real DB | < 5s | DynamoDB Local |
| E2E | Full user flows | < 30s | Full stack |

---

## NFR-TEST-001: Backend Unit Test Coverage

**Target:** Unit test coverage > 80% for backend Go code.

### Coverage Targets by Package

| Package | Minimum | Rationale |
|---------|---------|-----------|
| `internal/auth` | 85% | Security-critical authentication |
| `internal/api` | 80% | Request handling and validation |
| `internal/middleware` | 80% | Cross-cutting concerns |
| `internal/dynamo` | 75% | Data access layer |
| `internal/models` | 70% | Domain models |
| `internal/session` | 80% | Session management |
| `internal/health` | 75% | Health check logic |
| **Overall** | **80%** | Aggregate target |

### Measurement

```bash
# Generate coverage report
cd backend && go test -coverprofile=coverage.out ./...

# View overall coverage
go tool cover -func=coverage.out | grep total

# View per-package coverage
go tool cover -func=coverage.out

# HTML report for detailed analysis
go tool cover -html=coverage.out -o coverage.html
```

### CI Gate

```yaml
# .github/workflows/ci.yml
- name: Check Go coverage
  run: |
    go test -coverprofile=coverage.out ./...
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
      echo "Coverage $COVERAGE% is below 80% threshold"
      exit 1
    fi
```

### Test File Structure

```
backend/
├── internal/
│   ├── auth/
│   │   ├── middleware.go
│   │   └── middleware_test.go      # Unit tests
│   ├── api/
│   │   ├── handlers.go
│   │   └── handlers_test.go        # Unit tests
│   └── dynamo/
│       ├── repository.go
│       └── repository_test.go      # Unit tests (mocked)
└── test/
    └── integration/
        └── api_test.go             # Integration tests
```

### Test Template

```go
package auth

import (
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRequireAuth_WithoutCookie_ReturnsUnauthorized(t *testing.T) {
    // Arrange
    handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    req := httptest.NewRequest("GET", "/api/items", nil)
    w := httptest.NewRecorder()
    
    // Act
    handler.ServeHTTP(w, req)
    
    // Assert
    assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuth_WithValidCookie_CallsNext(t *testing.T) {
    // Arrange
    called := false
    handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        called = true
        w.WriteHeader(http.StatusOK)
    }))
    req := httptest.NewRequest("GET", "/api/items", nil)
    req.AddCookie(&http.Cookie{Name: "ds_session", Value: validToken})
    w := httptest.NewRecorder()
    
    // Act
    handler.ServeHTTP(w, req)
    
    // Assert
    require.True(t, called)
    assert.Equal(t, http.StatusOK, w.Code)
}
```

---

## NFR-TEST-002: Frontend Unit Test Coverage

**Target:** Frontend test coverage reaching 70% through phased approach.

### Phased Coverage Targets

| Phase | Timeline | Statements | Branches | Functions |
|-------|----------|------------|----------|-----------|
| Phase 1 (Baseline) | Current | 8% | 50% | 25% |
| Phase 2 | +2 sprints | 30% | 55% | 40% |
| Phase 3 | +4 sprints | 50% | 58% | 50% |
| Phase 4 (Target) | +6 sprints | 70% | 60% | 60% |

### Priority Test Areas

Focus test coverage on these high-value areas:

1. **Hooks** (`src/hooks/`)
   - `useAuth.tsx` - Authentication state
   - `useTheme.tsx` - Theme management
   - `useConsent.tsx` - Cookie consent

2. **API Client** (`src/api/`)
   - Request/response handling
   - Error handling
   - Token refresh

3. **Critical Components** (`src/components/`)
   - `ErrorBoundary.tsx` - Error handling
   - `CookieConsent.tsx` - GDPR compliance
   - `DSNav.tsx` - Navigation

### Vite Configuration

```typescript
// vite.config.ts
test: {
  coverage: {
    provider: 'v8',
    reporter: ['text', 'html', 'lcov'],
    thresholds: {
      statements: 8,    // Current baseline
      branches: 50,
      functions: 25,
      lines: 8,
    },
    exclude: [
      'node_modules/**',
      '**/*.d.ts',
      '**/*.config.*',
      '**/test/**',
    ],
  },
}
```

### Measurement

```bash
cd frontend

# Run tests with coverage
npm run test:coverage

# Output in terminal and coverage/ directory
```

### Test Template (React)

```typescript
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ThemeToggle } from './ThemeToggle';

describe('ThemeToggle', () => {
  it('renders light mode icon when theme is light', () => {
    render(<ThemeToggle theme="light" onToggle={vi.fn()} />);
    expect(screen.getByRole('button', { name: /switch to dark mode/i })).toBeInTheDocument();
  });

  it('calls onToggle when clicked', () => {
    const onToggle = vi.fn();
    render(<ThemeToggle theme="light" onToggle={onToggle} />);
    
    fireEvent.click(screen.getByRole('button'));
    
    expect(onToggle).toHaveBeenCalledTimes(1);
  });
});
```

---

## NFR-TEST-003: Integration Tests

**Target:** Integration tests for all API endpoints.

### Scope

Integration tests verify end-to-end behavior with real dependencies:

- DynamoDB operations via DynamoDB Local
- Authentication flows with mocked DSAccount
- Health check endpoint dependencies

### Directory Structure

```
backend/test/integration/
├── api_test.go           # API endpoint tests
├── fixtures.go           # Test data builders
├── setup_test.go         # Test setup/teardown
└── docker-compose.yml    # DynamoDB Local
```

### Setup

```yaml
# docker-compose.yml
version: '3.8'
services:
  dynamodb-local:
    image: amazon/dynamodb-local:latest
    ports:
      - "8000:8000"
    command: ["-jar", "DynamoDBLocal.jar", "-inMemory"]
```

### Running Integration Tests

```bash
# Start DynamoDB Local
docker-compose -f test/integration/docker-compose.yml up -d

# Run integration tests
cd backend && go test -v -tags=integration ./test/integration/...

# Cleanup
docker-compose -f test/integration/docker-compose.yml down
```

### Test Template

```go
//go:build integration

package integration

import (
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/stretchr/testify/assert"
)

func TestCreateItem_Integration(t *testing.T) {
    // Setup: real DynamoDB Local
    cleanup := setupTestTable(t)
    defer cleanup()
    
    server := setupTestServer(t)
    
    // Test: create item via API
    req := httptest.NewRequest("POST", "/api/items", 
        strings.NewReader(`{"name":"Test Item"}`))
    req.Header.Set("Content-Type", "application/json")
    req = addAuthCookie(req, testUser)
    
    w := httptest.NewRecorder()
    server.ServeHTTP(w, req)
    
    // Verify: status and response
    assert.Equal(t, http.StatusCreated, w.Code)
    
    // Verify: item exists in DynamoDB
    item := getItemFromDB(t, "test-item-id")
    assert.Equal(t, "Test Item", item.Name)
}
```

### Endpoint Coverage

| Endpoint | Test File | Status |
|----------|-----------|--------|
| GET /health | health_test.go | ✅ |
| POST /api/auth/login | auth_test.go | ✅ |
| GET /api/items | items_test.go | ⚠️ |
| POST /api/items | items_test.go | ⚠️ |
| PUT /api/items/:id | items_test.go | ⚠️ |
| DELETE /api/items/:id | items_test.go | ⚠️ |

---

## NFR-TEST-004: E2E Tests

**Target:** E2E tests for all critical user flows.

### Critical Flows

| Flow | Spec File | Priority |
|------|-----------|----------|
| Authentication | auth.spec.ts | Critical |
| Navigation | navigation.spec.ts | Critical |
| Theme switching | theme-i18n.spec.ts | High |
| Accessibility | accessibility.spec.ts | High |
| API Integration | api-integration.spec.ts | Critical |
| Error handling | error-handling.spec.ts | Medium |

### Directory Structure

```
frontend/e2e/
├── auth.spec.ts
├── navigation.spec.ts
├── theme-i18n.spec.ts
├── accessibility.spec.ts
├── api-integration.spec.ts
└── fixtures/
    ├── users.ts
    └── test-data.ts
```

### Running E2E Tests

```bash
cd frontend

# Run all E2E tests
npx playwright test

# Run specific test file
npx playwright test e2e/auth.spec.ts

# Run in headed mode (debug)
npx playwright test --headed

# Run with UI
npx playwright test --ui
```

### Test Template (Playwright)

```typescript
import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test('redirects to login when not authenticated', async ({ page }) => {
    await page.goto('/dashboard');
    await expect(page).toHaveURL(/.*login/);
  });

  test('shows user menu after login', async ({ page }) => {
    await page.goto('/');
    await page.click('[data-testid="login-button"]');
    
    // Complete SSO flow...
    
    await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
  });

  test('logout clears session', async ({ page }) => {
    // Login first
    await loginAsTestUser(page);
    
    // Logout
    await page.click('[data-testid="user-menu"]');
    await page.click('[data-testid="logout-button"]');
    
    // Verify redirect to login
    await expect(page).toHaveURL(/.*login/);
  });
});
```

### Accessibility Testing in E2E

```typescript
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.describe('Accessibility', () => {
  test('has no accessibility violations on home page', async ({ page }) => {
    await page.goto('/');
    
    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();
    
    expect(results.violations).toEqual([]);
  });
});
```

---

## NFR-TEST-005: All Tests Must Pass

**Target:** No deployment if any test fails.

### CI Pipeline Gates

```yaml
# .github/workflows/ci.yml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Backend unit tests
        run: cd backend && go test -race ./...
        
      - name: Backend integration tests
        run: |
          docker-compose -f test/docker-compose.yml up -d
          cd backend && go test -tags=integration ./test/integration/...
          
      - name: Frontend unit tests
        run: cd frontend && npm test
        
      - name: E2E tests
        run: cd frontend && npx playwright test
        
  deploy:
    needs: test  # Only runs if all tests pass
    runs-on: ubuntu-latest
    steps:
      - name: Deploy
        run: npm run deploy
```

### Branch Protection Rules

```yaml
# Repository settings
branch_protection:
  main:
    required_status_checks:
      strict: true
      contexts:
        - "Backend Tests"
        - "Frontend Tests"
        - "E2E Tests"
    required_reviews: 1
```

---

## Exception Process

When coverage requirements cannot be met, follow this process:

### 1. Document the Exception

```go
// COVERAGE-EXCEPTION: Repository methods use DynamoDB, tested via integration tests
// See: docs/requirements/nfr-testing.md#exception-process
// Tracking: #ISSUE-NUMBER
```

### 2. Acceptable Exception Reasons

| Reason | Example | Mitigation |
|--------|---------|------------|
| External Dependencies | DynamoDB client wrappers | Integration tests |
| Generated Code | CDK outputs, type definitions | Exclude from coverage |
| Infrastructure Code | CDK constructs | CDK synth validation |
| Trivial Code | Struct definitions, constants | N/A |

### 3. Request Exception Approval

1. Create a DSKanban issue with:
   - Package/file affected
   - Current vs. required coverage
   - Exception reason from table above
   - Mitigation plan

2. Assign to tech lead for review

3. If approved, update coverage exclusion:

**Go (coverprofile):**
```go
//go:build !coverage
```

**Vitest (vite.config.ts):**
```typescript
coverage: {
  exclude: ['**/generated/**']
}
```

### 4. Review Exceptions Quarterly

All coverage exceptions are reviewed quarterly. Exceptions that can be remediated should be converted to regular test coverage.

---

## Regression Testing

### Requirements

- Bug fixes MUST include regression test
- Test must fail before fix, pass after
- Test named descriptively: `TestIssue123_NullPointerOnEmptyInput`

### Template

```go
// Regression test for issue #123
// https://github.com/org/repo/issues/123
func TestIssue123_NullPointerOnEmptyInput(t *testing.T) {
    // This test verifies that empty input doesn't cause NPE
    // as reported in issue #123
    
    result, err := handler.Process("")
    
    require.NoError(t, err)
    assert.Empty(t, result)
}
```

---

## CI/CD Test Configuration

### Test Timeouts

| Test Type | Timeout |
|-----------|---------|
| Unit test | 30 seconds |
| Integration test | 5 minutes |
| E2E test | 10 minutes |
| Full suite | 30 minutes |

### Parallel Execution

```yaml
# CI configuration
test:
  parallel: 4
  shard: true
```

### Flaky Test Policy

- Flaky tests are bugs
- If test fails intermittently, fix or delete
- No `@flaky` annotations allowed
- Retry policy: none in CI (reveals flakes)

---

## Traceability

| Requirement | Implementation | Enforcement |
|-------------|----------------|-------------|
| NFR-TEST-001 | `go test -cover` | CI coverage gate |
| NFR-TEST-002 | `npm run test:coverage` | Vitest thresholds |
| NFR-TEST-003 | `test/integration/` | CI job |
| NFR-TEST-004 | `frontend/e2e/` | Playwright CI |
| NFR-TEST-005 | GitHub Actions | Branch protection |

### E2E to Requirement Traceability

E2E tests explicitly trace to requirement IDs, locking in functional contracts.

See [docs/reference/testing.md](../reference/testing.md) for:
- Test naming conventions (requirement ID in describe block)
- Coverage check script (`go run ./tools/cmd/check-requirement-coverage`)
- CI integration and enforcement
- Non-breaking change policy

---

*Last updated: 2026-03-22*
