# Maturity Schema Documentation

Machine-readable schema for automated maturity assessment.

**File:** `maturity.yaml` (repo root)  
**Consumer:** `assess-maturity` tool (GoTools)

---

## Schema Structure

```yaml
schema_version: 1          # Schema version (integer)
current_level: 0           # App's current maturity level (0-4)
target_level: 3            # Target maturity level

levels:
  0:
    name: "Prototype"
    description: "Brief description"
    checks: []             # List of check objects
```

### Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `schema_version` | integer | ✅ | Schema version for compatibility |
| `current_level` | integer | ✅ | App's assessed maturity level (0-4) |
| `target_level` | integer | ✅ | Target maturity level |
| `levels` | map | ✅ | Level definitions keyed by level number |

### Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Human-readable level name |
| `description` | string | ❌ | Brief description of level |
| `checks` | array | ✅ | List of check objects |

---

## Check Types

### `coverage`

Verify test coverage meets minimum threshold.

```yaml
- type: coverage
  target: backend    # backend | frontend
  min: 80            # Minimum percentage (0-100)
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `target` | string | ✅ | Component to check: `backend` or `frontend` |
| `min` | integer | ✅ | Minimum coverage percentage |

**Implementation:** Parses coverage reports from:
- Backend: `backend/coverage.out` (Go coverage format)
- Frontend: `frontend/coverage/coverage-summary.json` (Istanbul)

---

### `file_exists`

Verify a file or directory exists.

```yaml
- type: file_exists
  path: .github/workflows/ci.yml
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | ✅ | Path relative to repo root |

**Pass condition:** File/directory exists at specified path.

---

### `command`

Run an arbitrary shell command.

```yaml
- type: command
  name: "Backend builds"
  run: "cd backend && go build ./..."
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Human-readable check name |
| `run` | string | ✅ | Shell command to execute |

**Pass condition:** Command exits with code 0.

**Environment variables available:**
- `APP_NAME` — Application name
- `APP_URL` — Application URL (if deployed)
- `REPO_ROOT` — Repository root path

---

### `endpoint`

HTTP health check against a running endpoint.

```yaml
- type: endpoint
  name: "Health check"
  url: "/api/health"
  expect_status: 200
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Human-readable check name |
| `url` | string | ✅ | URL path (relative) or full URL |
| `expect_status` | integer | ❌ | Expected HTTP status (default: 200) |
| `expect_body` | string | ❌ | Substring expected in response body |

**Pass condition:** HTTP response matches expected status/body.

**Note:** Requires `APP_URL` environment variable for relative paths.

---

## Level Definitions

| Level | Name | Description |
|-------|------|-------------|
| 0 | Prototype | Works, no quality gates |
| 1 | Development | Unit tests, CI builds |
| 2 | Beta | E2E tests, auto-deploy to dev |
| 3 | Production | Full NFRs, monitoring, security review |
| 4 | Mature | Performance validated, DR tested |

---

## Usage

### assess-maturity Tool

```bash
# Assess current repo
go run ./tools/cmd/assess-maturity

# Assess specific level
go run ./tools/cmd/assess-maturity --level 2

# Output JSON
go run ./tools/cmd/assess-maturity --output json
```

### CI Integration

```yaml
# .github/workflows/ci.yml
- name: Maturity Check
  run: |
    go run ./tools/cmd/assess-maturity --level ${{ env.TARGET_LEVEL }}
```

---

## Schema Validation

The `assess-maturity` tool validates `maturity.yaml` on load:

1. `schema_version` must be supported
2. All required fields present
3. Check types are valid
4. Level numbers are integers 0-4

Invalid schema exits with error before running checks.

---

## Schema Version Compatibility

| Schema Version | Tool Version | Notes |
|----------------|--------------|-------|
| 1 | 0.1.0+ | Initial schema |

**Versioning rules:**

- **Patch version** (e.g., 1.0.1): Bug fixes, no schema changes
- **Minor version** (e.g., 1.1.0): New optional fields or check types (backwards compatible)
- **Major version** (e.g., 2.0.0): Breaking changes (field renames, removed types)

**Tool behavior:**

- `schema_version: 1` works with any tool that supports v1 schemas
- Unknown check types in a supported schema are skipped with a warning
- Unsupported schema version exits with an error before running checks

**Upgrading schemas:**

When a new schema version is released:

1. Update `schema_version` field in `maturity.yaml`
2. Migrate any deprecated fields per release notes
3. Test with `assess-maturity --dry-run`

---

## Adding Custom Checks

Apps can extend the default maturity checks with app-specific validations using the `command` check type.

### Example: Custom Security Check

```yaml
levels:
  3:
    name: Production
    checks:
      # ... standard checks ...
      - type: command
        name: "API authentication required"
        run: |
          # Verify all non-public endpoints require auth
          grep -r "RequireAuth" backend/internal/handlers/*.go | wc -l | \
            awk '{exit ($1 >= 5 ? 0 : 1)}'
```

### Example: Custom Performance Check

```yaml
levels:
  4:
    name: Mature
    checks:
      - type: command
        name: "Bundle size under limit"
        run: |
          cd frontend && pnpm build
          SIZE=$(du -sk dist | cut -f1)
          [ $SIZE -lt 300 ] || exit 1
```

### Best Practices for Custom Checks

1. **Use descriptive names** — Check names appear in output and CI logs
2. **Exit with proper codes** — 0 = pass, non-zero = fail
3. **Keep commands idempotent** — Safe to run multiple times
4. **Leverage environment variables** — Use `$APP_NAME`, `$APP_URL`, `$REPO_ROOT`
5. **Document in comments** — Explain what custom checks verify

### Environment Variables in Commands

| Variable | Description | Example |
|----------|-------------|---------|
| `APP_NAME` | Application name | `myapp` |
| `APP_URL` | Deployed application URL | `https://myapp.digistratum.com` |
| `REPO_ROOT` | Repository root path | `/home/user/repos/myapp` |
| `DS_ENVIRONMENT` | Current environment | `dev`, `prod` |

### Proposing New Check Types

If you need a reusable check type:

1. Open an issue in [GoTools](https://github.com/DigiStratum/GoTools)
2. Describe the use case and proposed schema
3. If approved, implement the handler and update this documentation

---

## See Also

- [MATURITY.md](MATURITY.md) — Human-readable maturity model
- [GoTools assess-maturity](https://github.com/DigiStratum/GoTools) — Tool implementation
