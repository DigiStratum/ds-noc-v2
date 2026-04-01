# AGENTS.md

## App Identity

**Name:** {{APP_DISPLAY_NAME}}
**Domain:** {{APP_DOMAIN}}
**Purpose:** [Describe what this app does]

## Maturity Level

**Current:** Level 1 (Development)
**Target:** Level 2 (Beta)

See [docs/MATURITY.md](docs/MATURITY.md) for level definitions.

**Check compliance:** `cd tools && go run ./cmd/assess-maturity`

### Implications at Level 1

| Aspect | Requirement | Status |
|--------|-------------|--------|
| Backend coverage | 40% minimum | 🟡 Track in CI |
| Frontend coverage | 30% minimum | 🟡 Track in CI |
| E2E tests | Not required | — |
| Security review | Not required | — |
| Monitoring | Not required | — |

### Acceptable at Level 1 (address before Level 2)

- [ ] Skip E2E tests for non-critical paths
- [ ] Basic error handling (stack traces OK in dev)
- [ ] Manual deployment to dev environment
- [ ] Minimal documentation

### Blockers for Level 2

- [ ] All FR-* requirements implemented with test coverage
- [ ] E2E tests for critical user paths
- [ ] Health endpoint returns 200
- [ ] HAL discovery returns all routes
- [ ] Automated deploy to dev on merge

## Standards & Conventions

**Read these first:**
- [docs/STANDARDS.md](docs/STANDARDS.md) — Tech stack, NFRs, code conventions (template-maintained)
- [docs/reference/architecture.md](docs/reference/architecture.md) — Common patterns and structure (template-maintained)
- [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md) — Requirements template (keep updated as app evolves)

## Multi-Tenant Isolation

All DS Ecosystem apps are multi-tenant. Data is isolated by tenant — cross-tenant access is denied by default.

### Tenant Model

```go
import "github.com/DigiStratum/ds-noc-v2/backend/pkg/tenant"

// Tenant types
tenant.TenantTypeUser  // "user" - personal workspace
tenant.TenantTypeOrg   // "org" - organization workspace

// Canonical format: "type:id" (e.g., "user:123" or "org:my-org")
t := tenant.Tenant{Type: tenant.TenantTypeUser, ID: "123"}
t.String() // "user:123"

// Parse from string
t, err := tenant.Parse("org:abc-def")

// Context helpers
ctx = tenant.SetTenant(ctx, t)
t := tenant.GetTenant(ctx)
t, err := tenant.RequireTenant(ctx) // errors if missing
```

### DynamoDB Key Patterns

All tenant-scoped data uses partition keys with tenant prefix:

```
pk: TENANT#user:123#ISSUE#456
pk: TENANT#org:my-org#PROJECT#abc
```

Use the helpers:

```go
// Build partition keys
pk := tenant.BuildPK(t, "ISSUE", issueID)    // TENANT#user:123#ISSUE#456
pk := tenant.BuildPK(t, "PROJECT", "abc", "MEMBER", userID)

// Parse partition keys (for validation)
parsedTenant, segments, err := tenant.ParsePK(pk)

// Validate key belongs to expected tenant (paranoid check)
err := tenant.ValidatePKBelongsToTenant(pk, expectedTenant)
```

### Middleware

Auth middleware extracts tenant from session/header and sets context:

```go
// In handler
t := tenant.GetTenant(r.Context())
if t.IsZero() {
    // Handle anonymous/no-tenant case
}

// For routes that REQUIRE tenant
mux.Handle("GET /api/issues", tenant.RequireTenantMiddleware(issueHandler))
```

### Audit Logging

Always include tenant in logs:

```go
// Structured logging
slog.Info("operation completed", tenant.LogAttrsFromContext(ctx)...)

// Audit events
tenant.LogAudit(ctx, tenant.AuditEvent{
    Action:     "create",
    Resource:   "issue",
    ResourceID: "123",
    Actor:      userID,
})

// Security: log cross-tenant access attempts (should never happen)
tenant.LogCrossTenantAttempt(ctx, requestedTenant, actualTenant, "issue", "123")
```

### Security Rules

1. **All data access must be tenant-scoped** — no exceptions
2. **Use `tenant.RequireTenant(ctx)`** in handlers that access data
3. **Use `tenant.BuildPK()`** for all DynamoDB keys — never build manually
4. **Validate keys with `tenant.ValidatePKBelongsToTenant()`** before returning data
5. **Log tenant on all audit events** — use `tenant.LogAudit()` or include `tenant.LogAttrsFromContext()`

## App-Specific Context

[Add app-specific domain knowledge, business rules, and customizations here]

## Before Every Commit

Run the pre-flight quality gate to catch CI failures locally:

```bash
cd tools && go run ./cmd/pre-flight
```

**Automate with a git hook:**

```bash
# Install pre-commit hook (runs pre-flight before each commit)
cd tools && go run ./cmd/pre-flight --install-hook

# To uninstall
cd tools && go run ./cmd/pre-flight --uninstall-hook

# To bypass for a single commit
git commit --no-verify -m "message"
```

**What it checks:**

| Category | Checks | Pass Criteria |
|----------|--------|---------------|
| Backend | `go build`, `golangci-lint`, `go test` | No errors, lint clean, tests pass |
| Frontend | `tsc`, `eslint`, `vitest` | No type errors, lint clean, tests pass |
| Template | Manifest coverage | All files in manifest or app-owned dirs |
| API | HAL compliance | All routes in discovery, proper format |
| Builds | Backend binary, frontend dist, CDK synth | All builds succeed |

**Options:**
- `--fix` — Auto-fix lint issues where possible
- `--verbose` — Show output for passing checks

### Manual Checklist

Before committing, verify:

- [ ] New endpoints added via `add-endpoint` (not manually)
- [ ] New routes registered in `discovery.go` 
- [ ] Frontend uses `useHALNavigation()` (no hardcoded paths)
- [ ] Tests cover new functionality
- [ ] Migrations versioned properly (if schema changes)

## Adding Features Workflow

Use the scaffolding tools for consistent structure and automatic wiring:

### 1. Backend First

```bash
# Create endpoint + handler + discovery link (handler name auto-derived)
cd tools && go run ./cmd/add-endpoint POST /api/products

# If storing data, create model
go run ./cmd/add-model Product --pk ProductID:string

# If complex logic, create service
go run ./cmd/add-service Product --dep ProductRepository
```

### 2. Frontend Second

```bash
# Create page with route
go run ./cmd/add-page ProductList --route /products --with-data

# Create reusable components
go run ./cmd/add-component ProductCard

# Create data-fetching hook
go run ./cmd/add-hook useProducts --with-fetch
```

### 3. Full Vertical Slice (Shortcut)

For standard CRUD features:

```bash
go run ./cmd/add-feature Product
```

This creates: endpoint, service, page, hook, and types in one command.

### 4. Verify Before Commit

```bash
cd tools && go run ./cmd/pre-flight
```

## Scaffolding Tools

**⚠️ NEVER manually create handlers, models, pages, or hooks from scratch.**

Use the scaffolding tools in `tools/`. These automatically download dependencies via Go modules at dev-time — no pre-installed binaries required.

### Backend Scaffolding

```bash
# API endpoint (handler + route + HAL link + test)
# Handler name is automatically derived from METHOD + PATH:
#   GET /api/items → ListItems
#   POST /api/items → CreateItem  
#   GET /api/items/{id} → GetItem
go run ./tools/cmd/add-endpoint GET /api/items
go run ./tools/cmd/add-endpoint POST /api/items
go run ./tools/cmd/add-endpoint GET /api/items/{id}

# DynamoDB model (struct + repository + CRUD)
go run ./tools/cmd/add-model Product --pk ProductID:string --gsi ByCategory:CategoryID:string

# Business logic service (interface + DI)
go run ./tools/cmd/add-service Order --dep OrderRepository --dep Logger

# HTTP middleware (auth, logging, rate-limit, cors, custom)
go run ./tools/cmd/add-middleware RateLimit --type rate-limit
go run ./tools/cmd/add-middleware TenantContext --type custom
```

### Frontend Scaffolding

```bash
# React page (component + route registration)
go run ./tools/cmd/add-page Dashboard --route /dashboard --with-data

# Reusable component (props + test + optional Storybook)
go run ./tools/cmd/add-component Button --storybook
go run ./tools/cmd/add-component DataTable

# Custom hook (types + test)
go run ./tools/cmd/add-hook useAuth --with-fetch
go run ./tools/cmd/add-hook useToggle --with-state
```

### Full Vertical Slice

```bash
# Creates: endpoint + service + page + hook + types
go run ./tools/cmd/add-feature Product
```

### Environment Variables

```bash
# Adds to CDK, .env.example, and README
go run ./tools/cmd/add-env-var DS_FEATURE_FLAG --description "Enable new feature" --default false
```

### OpenAPI Sync

```bash
# Generate OpenAPI spec from HAL discovery
go run ./tools/cmd/sync-openapi https://api.example.com/api/discovery --output docs/api/openapi.yaml
```

### After Scaffolding

1. Implement business logic in generated files
2. Add route to `backend/cmd/api/main.go` (endpoint scaffolder shows exact code)
3. Write/expand tests
4. Run: `go build ./... && go test ./...`

## Utility Tools

```bash
# Assess maturity level compliance
go run ./cmd/assess-maturity              # All levels
go run ./cmd/assess-maturity --level 2    # Specific level
go run ./cmd/assess-maturity --output json # JSON for CI

# Build backend with version info
go run ./cmd/build-backend [output-path]

# Verify manifest coverage
go run ./cmd/check-manifest [--strict]

# Check E2E requirement coverage
go run ./cmd/check-requirement-coverage [--strict] [--json]

# Verify HAL/HATEOAS compliance
go run ./cmd/verify-hal-compliance [--frontend]

# Pull template updates (run from app directory)
go run ./cmd/update-from-template [--dry-run] [--template-path /path/to/template]
```

**Note:** All tools run from the `tools/` directory: `cd tools && go run ./cmd/<tool>`

## HAL/HATEOAS Compliance

This app uses HAL+JSON for HATEOAS-driven API discovery.

**Rules:**
- All API endpoints must be in `discovery.go`
- All responses must include `_links.self`
- Use `Content-Type: application/hal+json`
- Frontend uses `useHALNavigation()` hook — no hardcoded API paths

**CI enforces this.** PRs will fail if routes exist without discovery links.

## Transaction Logging

All apps include automatic transaction logging to `/ds/ecosystem/transactions` CloudWatch log group.

### Automatic Fields (captured by middleware)
- Request method, path, resource ID
- Response status, duration, bytes
- Correlation ID, session ID

### Setting Data Type
Handlers should set the response data type for better metrics:

```go
func (h *Handler) GetWidget(w http.ResponseWriter, r *http.Request) {
    if txn := txnlog.FromContext(r.Context()); txn != nil {
        txn.SetDataType("Widget")
    }
    // ...
}
```

### Environment Variables
| Variable | Required | Description |
|----------|----------|-------------|
| TXNLOG_GROUP | Yes | CloudWatch log group path |
| APP_ID | Yes | App identifier (e.g., "dskanban") |
| APP_NAME | Yes | Human-readable name |
| ENV | Yes | Environment (prod/staging/dev) |

### Local Development
When TXNLOG_GROUP is unset or CloudWatch is unavailable, txnlog logs warnings but does not fail requests.

## Database Migrations

Use the `db-migrate` tool for versioned DynamoDB schema and data migrations.

### When to Create a Migration

- Adding a new table
- Adding a Global Secondary Index (GSI)
- Backfilling data to new fields
- Transforming field formats
- Any schema change that needs to be tracked and reproducible

**Do NOT use migrations for:** Normal CRUD operations, temporary scripts, or one-off fixes.

### Structure

```
migrations/
├── 001_create_products.go      # Table creation
├── 002_add_category_gsi.go     # Add GSI
├── 003_backfill_created_at.go  # Data backfill
└── registry.go                 # Register all migrations
```

### Writing Migrations

Each migration implements the `Migration` interface:

```go
package migrations

import (
    "context"
    "github.com/DigiStratum/GoTools/codegen/dbmigrate"
)

type M001_CreateProducts struct{}

func (m *M001_CreateProducts) Version() string { return "001" }
func (m *M001_CreateProducts) Name() string    { return "create_products_table" }

func (m *M001_CreateProducts) Up(ctx context.Context, db dbmigrate.DynamoDBClient) error {
    return db.CreateTable(ctx, &dbmigrate.CreateTableInput{
        TableName:        "products",
        PartitionKey:     "product_id",
        PartitionKeyType: "S",
        BillingMode:      "PAY_PER_REQUEST",
    })
}

func (m *M001_CreateProducts) Down(ctx context.Context, db dbmigrate.DynamoDBClient) error {
    return db.DeleteTable(ctx, "products")
}
```

### Registry Pattern

Create a `registry.go` to list all migrations:

```go
package migrations

import "github.com/DigiStratum/GoTools/codegen/dbmigrate"

func All() []dbmigrate.Migration {
    return []dbmigrate.Migration{
        &M001_CreateProducts{},
        &M002_AddCategoryGSI{},
        &M003_BackfillCreatedAt{},
    }
}
```

### Testing Locally (DynamoDB Local)

```bash
# Start DynamoDB Local
docker run -p 8000:8000 amazon/dynamodb-local

# Run migrations
DYNAMODB_ENDPOINT=http://localhost:8000 go run ./tools/cmd/db-migrate up

# Check status
DYNAMODB_ENDPOINT=http://localhost:8000 go run ./tools/cmd/db-migrate status
```

### Running in Production

```bash
# Preview changes (dry run)
go run ./tools/cmd/db-migrate up --dry-run --env prod

# Apply migrations
go run ./tools/cmd/db-migrate up --env prod

# Check status
go run ./tools/cmd/db-migrate status --env prod
```

### Rollback Procedures

```bash
# Rollback last migration
go run ./tools/cmd/db-migrate down --env prod
```

**⚠️ Warning:** Not all migrations are safely reversible:
- **GSI deletions** can cause data loss
- **Data backfills** may not have clean rollbacks
- **Field transformations** may lose precision

Always test rollbacks locally before relying on them in production.

### Best Practices

1. **Version prefix**: Use `001_`, `002_` etc. for ordering
2. **Descriptive names**: `add_category_gsi`, not `migration2`
3. **Idempotent Up**: Check if already applied before acting
4. **Rate limit backfills**: Use `RateLimit` config to avoid throttling
5. **Test Down**: Every Up should have a working Down
6. **Document destructive changes**: If Down loses data, say so

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DS_ENVIRONMENT` | Environment (dev, stage, prod) |
| `DS_TABLE_PREFIX` | Prefix for table names |
| `DYNAMODB_ENDPOINT` | Custom endpoint (DynamoDB Local) |

### CLI Reference

```bash
go run ./tools/cmd/db-migrate up                    # Apply pending
go run ./tools/cmd/db-migrate up --dry-run          # Preview
go run ./tools/cmd/db-migrate down                  # Rollback last
go run ./tools/cmd/db-migrate status                # Show status
go run ./tools/cmd/db-migrate up --env prod         # Production
go run ./tools/cmd/db-migrate up --endpoint http://localhost:8000  # Local
```

## Directory Structure

```
backend/
  cmd/api/main.go     # Route registration (Lambda entry point)
  internal/
    handlers/         # Your endpoint handlers (add via script)
    [template-owned]  # auth, discovery, hal, health, middleware, session

frontend/
  src/
    app/              # App-specific: config, Layout, pages, features
    [template-owned]  # api, components, hooks

infra/                # CDK stack (template-owned)
docs/                 # Standards, architecture (template-maintained)
tools/                # Go scaffolding tools (template-maintained)
```

## Build & Test

```bash
# Backend
cd backend && go build ./... && go test ./...

# Frontend
cd frontend && pnpm install && pnpm build && pnpm test

# Infra
cd infra && pnpm install && npx cdk synth
```

## Deployment Workflow

### Branch Strategy

| Branch | Environment | Auto-deploy? | Notes |
|--------|-------------|--------------|-------|
| develop | dev | Yes | Default work branch |
| main | prod | Yes | Protected, PR required |

### Workflow

1. All work happens on `develop` branch (or feature branches → develop)
2. Push to develop triggers CI/CD:
   - Build and test
   - Deploy to dev environment
   - Health check verification
3. On success, auto-merge workflow:
   - Creates PR: develop → main
   - Auto-merges when CI passes
4. Push to main triggers prod deploy:
   - Deploy to production
   - Health check verification

### Agent Deployment Steps

```bash
# Complete work on develop
git checkout develop
# ... make changes ...
git commit -m "feat: description"
git push origin develop

# CI/CD handles the rest:
# - Dev deploy + health check
# - Auto PR + merge to main
# - Prod deploy + health check
```

### Rollback

If prod health check fails:
1. Revert the merge commit on main
2. Or: manually deploy previous Lambda version

## Template Updates

This app was created from `ds-app-template`. To pull template updates:

```bash
go run ./tools/cmd/update-from-template --dry-run  # Preview changes
go run ./tools/cmd/update-from-template            # Apply changes
```

**Template-owned paths** (auto-updated): see `.template-manifest`
**App-owned paths** (never touched): `AGENTS.md`, `README.md`, `frontend/src/app/`, `backend/internal/handlers/`

### Customizing Template Files (`.template-overrides`)

If you need to customize a template-owned file (e.g., custom API client, modified shared component), add it to `.template-overrides` in your app root:

```bash
# Example .template-overrides
frontend/src/api/client.ts
frontend/src/api/index.ts
frontend/src/components/AuthShell.tsx
```

Files listed here will be skipped during template updates. The dry-run output shows `[override]` for these files.

**When to use:**
- You've customized a shared component for your app
- You need app-specific API client configuration
- Template hooks don't fit your use case

See [docs/updating-apps.md](docs/updating-apps.md) for full documentation.
