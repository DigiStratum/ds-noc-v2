# Testing Guide

> Practical testing guidance for DS applications.
> For measurable NFR targets and enforcement, see [NFR-Testing](../requirements/nfr-testing.md).

---

## Test Pyramid

```
        /\
       /  \     E2E Tests (Playwright)
      /----\    - Critical user flows
     /      \   - Run on deploy
    /--------\
   /          \ Integration Tests
  /            \ - API contracts
 /              \ - Service interactions
/----------------\
     Unit Tests
   - Functions, components
   - Fast, isolated
```

### Layer Responsibilities

| Layer | Scope | Speed | Dependencies |
|-------|-------|-------|--------------|
| Unit | Single function/component | < 100ms | Mocked |
| Integration | Multiple components, real DB | < 5s | DynamoDB Local |
| E2E | Full user flows | < 30s | Full stack |

---

## Running Tests

```bash
# Backend - all tests
cd backend && go test ./...

# Backend - with coverage
cd backend && go test -coverprofile=coverage.out ./...

# Frontend - all tests  
cd frontend && pnpm test

# Frontend - with coverage
cd frontend && pnpm test --coverage

# E2E tests
cd e2e && pnpm test
```

---

## Test Naming

### Go

```go
func TestHandleUserGet_ValidUser(t *testing.T) { ... }
func TestHandleUserGet_NotFound(t *testing.T) { ... }
```

Format: `Test<Function>_<Scenario>`

### TypeScript

```typescript
describe('UserCard', () => {
  it('displays user name', () => { ... });
  it('calls onEdit when button clicked', () => { ... });
});
```

---

## Bug Fix Regression Tests

**All bug fixes must include a regression test.**

The test should:
1. Fail before the fix (reproduces the bug)
2. Pass after the fix
3. Reference the issue number in test name or comment

```go
// Regression test for #1234
func TestHandleUserGet_NilPointerFix(t *testing.T) { ... }
```

If a test cannot be added, document the reason in the PR.

---

## Test Data

### Fixtures
- Backend: `testdata/` directory
- Frontend: `__fixtures__/` or co-located

### Mocking
- Backend: Interface-based mocking
- Frontend: MSW for API mocking

---

## E2E Requirement Traceability

E2E tests trace to requirements, locking in functional contracts.

### Test Naming Convention

E2E tests reference requirement IDs in their `describe` blocks:

```typescript
// Good: Explicit requirement ID in describe block
test.describe('FR-AUTH-001: Users authenticate via DSAccount SSO', () => {
  test('should redirect unauthenticated user to SSO', async ({ page }) => {
    // ...
  });
});

// Bad: No requirement ID
test.describe('Authentication', () => {
  test('login works', async ({ page }) => {
    // Which requirement? Not traceable!
  });
});
```

### Test File Organization

Test files mirror requirement categories:

| Test File | Requirements |
|-----------|--------------|
| `e2e/auth.spec.ts` | `FR-AUTH-*` |
| `e2e/navigation.spec.ts` | `FR-NAV-*` |
| `e2e/tenant.spec.ts` | `FR-TENANT-*` |
| `e2e/theme.spec.ts` | `FR-THEME-*` |
| `e2e/i18n.spec.ts` | `FR-I18N-*` |
| `e2e/accessibility.spec.ts` | `NFR-A11Y-*` |
| `e2e/performance.spec.ts` | `NFR-PERF-*` |

### Coverage Check

```bash
# Run coverage check
go run ./tools/cmd/check-requirement-coverage

# Strict mode (fail if gaps exist)
go run ./tools/cmd/check-requirement-coverage --strict

# JSON output for CI integration
go run ./tools/cmd/check-requirement-coverage --json
```

Sample output:
```
═══════════════════════════════════════════════════════════════
           E2E Test to Requirement Traceability Report         
═══════════════════════════════════════════════════════════════

  Total requirements:    28
  Tested requirements:   22
  Skipped tests:         1
  Untested requirements: 6
  Orphaned tests:        0

  Coverage: [████████████████████████████░░░░░░░░░░░░] 78.6%
```

### Skipping Tests

If a requirement cannot be tested, use `test.skip`:

```typescript
test.describe('FR-I18N-002: Dynamic content translated', () => {
  test.skip('should translate dynamic content', async ({ page }) => {
    // Skipped until dynamic translation is implemented
  });
});
```

---

## Adding New Requirements

1. **Add to REQUIREMENTS.md** with new ID (never reuse IDs)
2. **Create E2E test** with requirement ID in describe block
3. **Run coverage check** to verify linkage
4. **Commit both** together

---

## Deprecating Requirements

Requirements can be deprecated but **never deleted**:

1. Mark in REQUIREMENTS.md:
   ```markdown
   - ❌ DEPRECATED (2026-03): FR-LEGACY-001 - Replaced by FR-AUTH-001
   ```

2. Keep the E2E test with a skip:
   ```typescript
   test.describe.skip('FR-LEGACY-001: Old auth flow', () => {
     // Deprecated 2026-03, replaced by FR-AUTH-001
   });
   ```

---

## Best Practices

### DO
- One `describe` block per requirement
- Multiple `test` cases under same requirement for different scenarios
- Include requirement ID AND human-readable description in describe
- Keep tests isolated and deterministic

### DON'T
- Have tests without requirement IDs
- Put multiple requirement IDs in one describe block
- Delete tests without approval process
- Reuse deprecated requirement IDs

---

## CI Integration

Tests run on:
- PR creation/update
- Push to main
- Pre-deployment

Failures block merge/deploy.

### Flaky Test Policy

- Flaky tests are bugs
- If test fails intermittently, fix or delete
- No `@flaky` annotations allowed

---

## Related Documentation

- [NFR: Testing](../requirements/nfr-testing.md) — Coverage targets, enforcement, exception process
- [Code Conventions](conventions.md) — Test file organization

---

*Last updated: 2026-03-23*
