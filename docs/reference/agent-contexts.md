# Agent Contexts

> Role-specific context for AI agents working on DS apps. Load the relevant section when performing domain-specific work.

---

## Overview

| Role | Load When |
|------|-----------|
| [Backend](#backend) | Go handlers, middleware, storage, API endpoints |
| [Frontend](#frontend) | React components, pages, hooks, frontend features |
| [Infrastructure](#infrastructure) | CDK, deployments, environment config, monitoring |
| [Security](#security) | Authentication, authorization, input validation, security fixes |
| [Testing](#testing) | Implementing tests, fixing test failures, adding coverage |

---

## Backend

### Quick Reference

| Aspect | Standard |
|--------|----------|
| Language | Go 1.21+ |
| HTTP | net/http (stdlib) |
| Response format | HAL+JSON |
| Storage | DynamoDB (prod), SQLite (dev) |
| Auth | SSO via DSAccount |

### Directory Structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go            # Lambda entry, route registration
│
├── internal/                  # Non-exported packages
│   ├── handlers/              # App-owned: your endpoint handlers
│   │   └── items.go
│   │
│   ├── auth/                  # Template: SSO validation
│   ├── buildinfo/             # Template: version info
│   ├── discovery/             # Template: HAL discovery endpoint
│   ├── hal/                   # Template: HAL response helpers
│   ├── health/                # Template: health check
│   ├── middleware/            # Template: common middleware
│   └── session/               # Template: session management
│
├── pkg/                       # Exported packages (rare)
│
└── go.mod
```

#### What Goes Where

| Location | Purpose | Ownership |
|----------|---------|-----------|
| `internal/handlers/` | Your API handlers | App |
| `internal/storage/` | Storage interface/impl | App |
| `internal/domain/` | Business logic | App |
| `internal/auth/` | SSO middleware | Template |
| `internal/hal/` | HAL response helpers | Template |
| `cmd/api/main.go` | Route registration | Template |

### Adding New Endpoints

**⚠️ NEVER manually edit main.go or discovery.go.**

Use the scaffolding script:

```bash
./go run ./tools/cmd/add-endpoint --name items --path /api/items --method GET
./go run ./tools/cmd/add-endpoint --name item --path "/api/items/{id}" --method GET --auth required
./go run ./tools/cmd/add-endpoint -n create-item -p /api/items -m POST -a required -d "Create new item"
```

The script:
1. Creates handler file in `backend/internal/handlers/`
2. Creates test file stub
3. Adds link to `backend/internal/discovery/discovery.go`
4. Adds relation constant to `backend/internal/hal/rels.go`
5. Tells you where to add the route in `main.go`

### Handler Patterns

#### Basic Handler

```go
package handlers

import (
    "encoding/json"
    "net/http"
    
    "myapp/internal/hal"
    "myapp/internal/storage"
)

type ItemHandler struct {
    store storage.ItemStore
}

func NewItemHandler(store storage.ItemStore) *ItemHandler {
    return &ItemHandler{store: store}
}

func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    items, err := h.store.ListItems(ctx)
    if err != nil {
        hal.RespondError(w, r, http.StatusInternalServerError, "failed to list items")
        return
    }
    
    hal.Respond(w, r, http.StatusOK, hal.Resource{
        Links: hal.Links{
            "self": hal.Link{Href: "/api/items"},
        },
        Data: map[string]interface{}{
            "items": items,
        },
    })
}
```

#### Handler with Path Parameters

```go
func (h *ItemHandler) Get(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")  // or mux.Vars(r)["id"]
    if id == "" {
        hal.RespondError(w, r, http.StatusBadRequest, "id required")
        return
    }
    
    item, err := h.store.GetItem(r.Context(), id)
    if err != nil {
        if errors.Is(err, storage.ErrNotFound) {
            hal.RespondError(w, r, http.StatusNotFound, "item not found")
            return
        }
        hal.RespondError(w, r, http.StatusInternalServerError, "failed to get item")
        return
    }
    
    hal.Respond(w, r, http.StatusOK, hal.Resource{
        Links: hal.Links{
            "self": hal.Link{Href: fmt.Sprintf("/api/items/%s", id)},
        },
        Data: item,
    })
}
```

#### Handler with Request Body

```go
type CreateItemRequest struct {
    Title       string `json:"title"`
    Description string `json:"description"`
}

func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateItemRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        hal.RespondError(w, r, http.StatusBadRequest, "invalid JSON")
        return
    }
    
    // Validate
    if err := h.validateCreate(req); err != nil {
        hal.RespondError(w, r, http.StatusBadRequest, err.Error())
        return
    }
    
    // Get tenant from session
    session := auth.SessionFromContext(r.Context())
    
    item, err := h.store.CreateItem(r.Context(), storage.CreateItemInput{
        TenantID:    session.TenantID,
        Title:       req.Title,
        Description: req.Description,
    })
    if err != nil {
        hal.RespondError(w, r, http.StatusInternalServerError, "failed to create item")
        return
    }
    
    hal.Respond(w, r, http.StatusCreated, hal.Resource{
        Links: hal.Links{
            "self": hal.Link{Href: fmt.Sprintf("/api/items/%s", item.ID)},
        },
        Data: item,
    })
}

func (h *ItemHandler) validateCreate(req CreateItemRequest) error {
    if req.Title == "" {
        return errors.New("title is required")
    }
    if len(req.Title) > 200 {
        return errors.New("title too long (max 200)")
    }
    return nil
}
```

### Middleware Usage

#### Auth Middleware

```go
// In main.go route registration
router.With(auth.RequireAuth).Get("/api/items", handler.List)
router.With(auth.RequireAuth).Post("/api/items", handler.Create)

// Public endpoints (no auth)
router.Get("/api/health", health.Handler)
router.Get("/api/discovery", discovery.Handler)
```

#### Accessing Session in Handlers

```go
func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
    session := auth.SessionFromContext(r.Context())
    
    // Always scope queries to tenant
    items, err := h.store.ListItems(r.Context(), session.TenantID)
    // ...
}
```

### Database Access Patterns

#### Storage Interface

```go
// internal/storage/interface.go
package storage

import "context"

type ItemStore interface {
    ListItems(ctx context.Context, tenantID string) ([]Item, error)
    GetItem(ctx context.Context, id string) (*Item, error)
    CreateItem(ctx context.Context, input CreateItemInput) (*Item, error)
    UpdateItem(ctx context.Context, id string, input UpdateItemInput) (*Item, error)
    DeleteItem(ctx context.Context, id string) error
}

type CreateItemInput struct {
    TenantID    string
    Title       string
    Description string
}
```

#### DynamoDB Implementation

```go
// internal/dynamo/items.go
package dynamo

import (
    "context"
    
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type ItemStore struct {
    client    *dynamodb.Client
    tableName string
}

func (s *ItemStore) ListItems(ctx context.Context, tenantID string) ([]storage.Item, error) {
    // Always query by partition key (tenant)
    input := &dynamodb.QueryInput{
        TableName:              aws.String(s.tableName),
        KeyConditionExpression: aws.String("tenant_id = :tenant"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":tenant": &types.AttributeValueMemberS{Value: tenantID},
        },
    }
    
    result, err := s.client.Query(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("query items: %w", err)
    }
    
    var items []storage.Item
    if err := attributevalue.UnmarshalListOfMaps(result.Items, &items); err != nil {
        return nil, fmt.Errorf("unmarshal items: %w", err)
    }
    
    return items, nil
}
```

#### SQLite Implementation (Local Dev)

```go
// internal/storage/sqlite.go
package storage

import "database/sql"

type SQLiteStore struct {
    db *sql.DB
}

func (s *SQLiteStore) ListItems(ctx context.Context, tenantID string) ([]Item, error) {
    rows, err := s.db.QueryContext(ctx, 
        "SELECT id, title, description FROM items WHERE tenant_id = ?",
        tenantID,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var items []Item
    for rows.Next() {
        var item Item
        if err := rows.Scan(&item.ID, &item.Title, &item.Description); err != nil {
            return nil, err
        }
        items = append(items, item)
    }
    return items, nil
}
```

**Critical:** If you modify storage behavior, **you must update both** `internal/dynamo/` (production) and `internal/storage/sqlite.go` (local dev).

### Error Handling

```go
// Use hal.RespondError for consistent format
hal.RespondError(w, r, http.StatusNotFound, "item not found")

// Wrap errors with context
if err != nil {
    return fmt.Errorf("create item: %w", err)
}

// Check specific errors
if errors.Is(err, storage.ErrNotFound) {
    hal.RespondError(w, r, http.StatusNotFound, "item not found")
    return
}

// Logging errors
slog.Error("failed to create item",
    "error", err,
    "tenant_id", session.TenantID,
    "request_id", middleware.RequestIDFromContext(r.Context()),
)
```

### HAL/HATEOAS Compliance

Every response must include `_links.self`. Every route in `main.go` must have a corresponding link in `discovery.go` (CI enforced).

```go
hal.Respond(w, r, http.StatusOK, hal.Resource{
    Links: hal.Links{
        "self": hal.Link{Href: "/api/items"},
        "ds:create-item": hal.Link{Href: "/api/items", Method: "POST"},
    },
    Data: items,
})
```

### Backend Common Mistakes

| ❌ Bad | ✅ Good |
|--------|---------|
| No tenant scoping | Always scope to `session.TenantID` |
| Exposing `err.Error()` | Generic message, log internally |
| Missing input validation | Validate before processing |
| Only handling happy path | Handle all error cases |
| Manual route registration | Use scaffolding script |

### Build Commands

```bash
cd backend
go build ./...                                    # Build
STORAGE_TYPE=sqlite go run ./cmd/api              # Run locally
GOOS=linux GOARCH=arm64 go build -o bootstrap ./cmd/lambda  # Build for Lambda
go test ./...                                     # Test
go test -coverprofile=coverage.out ./...          # Coverage
```

---

## Frontend

### Quick Reference

| Aspect | Standard |
|--------|----------|
| Framework | React 18, TypeScript, Vite |
| Styling | TailwindCSS, shadcn/ui |
| State | React Query (server), React hooks (local) |
| Testing | Vitest, React Testing Library |
| A11y | WCAG 2.1 AA |

### Directory Structure

```
frontend/src/
├── App.tsx                    # Entry point, routing
├── main.tsx                   # React DOM render
├── index.css                  # Global styles (Tailwind)
│
├── app/                       # App-owned (your code)
│   ├── config/                # App configuration
│   ├── pages/                 # Route components
│   ├── features/              # Feature modules
│   └── Layout.tsx             # App shell wrapper
│
├── components/                # Template-owned shared components
├── hooks/                     # Template-owned hooks
└── api/                       # Template-owned API client
```

#### What Goes Where

| Location | Purpose | Ownership |
|----------|---------|-----------|
| `src/app/pages/` | Route components | App |
| `src/app/features/` | Feature modules | App |
| `src/components/` | Shared UI components | Template |
| `src/hooks/` | Shared hooks | Template |
| `src/api/` | API client | Template |

### Component Patterns

#### Basic Component

```tsx
import { type FC } from 'react';

interface ItemCardProps {
  title: string;
  description?: string;
  onSelect: (id: string) => void;
}

export const ItemCard: FC<ItemCardProps> = ({ title, description, onSelect }) => {
  return (
    <article 
      className="p-4 border rounded-lg hover:border-primary"
      onClick={() => onSelect(title)}
    >
      <h3 className="font-medium">{title}</h3>
      {description && (
        <p className="text-sm text-muted-foreground">{description}</p>
      )}
    </article>
  );
};
```

#### Component with Loading/Error States

```tsx
import { useItems } from '../hooks/useItems';

export const ItemList: FC = () => {
  const { data: items, isLoading, error } = useItems();

  if (isLoading) return <Skeleton className="h-12 w-full" />;
  if (error) return <Alert variant="destructive">Failed to load items.</Alert>;
  if (!items?.length) return <p>No items yet.</p>;

  return (
    <ul className="space-y-2">
      {items.map(item => <li key={item.id}><ItemCard {...item} /></li>)}
    </ul>
  );
};
```

### State Management

#### Server State (React Query)

```tsx
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

export function useItems() {
  return useQuery({
    queryKey: ['items'],
    queryFn: () => api.get('/api/items'),
  });
}

export function useCreateItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateItemInput) => api.post('/api/items', data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['items'] }),
  });
}
```

#### Session State (@digistratum/ds-core)

```tsx
import { useSessionData } from '@digistratum/ds-core';

function MyComponent() {
  const { data, setData, clearData } = useSessionData<MyData>('my-key');
}
```

### HAL/HATEOAS Navigation

**Never hardcode API paths.** Use the HAL navigation hook:

```tsx
import { useHALNavigation } from '@/hooks/useHALNavigation';

function Dashboard() {
  const { navigate, getLink } = useHALNavigation();
  const createItemUrl = getLink('ds:create-item');
  
  const handleCreate = async (data: ItemInput) => {
    await navigate('ds:create-item', { method: 'POST', body: data });
  };
}
```

### Accessibility Requirements

| Criterion | Requirement |
|-----------|-------------|
| Color contrast | 4.5:1 text, 3:1 large text |
| Keyboard | All functionality via keyboard |
| Focus | Visible focus indicators |
| Screen reader | Semantic HTML, ARIA where needed |

```tsx
// Accessible patterns
<nav aria-label="Main"><ul><li><a href="/home">Home</a></li></ul></nav>
<button aria-label="Delete item"><TrashIcon aria-hidden="true" /></button>
<label htmlFor="email">Email</label><input id="email" type="email" required />
<div aria-busy="true" aria-live="polite">Loading...</div>
```

### Frontend Common Mistakes

| ❌ Bad | ✅ Good |
|--------|---------|
| Hardcoding API paths | Use `getLink()` from HAL |
| Missing loading/error states | Always handle isLoading, error, empty |
| Missing key props | Always provide unique keys |
| Non-semantic HTML (`<div onClick>`) | Use proper elements (`<button>`) |
| Missing form labels | Always label inputs |
| Direct DOM manipulation | Use React state |

### Build Commands

```bash
cd frontend
pnpm install        # Install
pnpm dev            # Dev server
pnpm build          # Production build
pnpm test           # Tests
pnpm test:coverage  # Coverage
pnpm typecheck      # Type checking
pnpm lint           # Linting
```

---

## Infrastructure

### Quick Reference

| Aspect | Technology |
|--------|------------|
| IaC | AWS CDK (TypeScript) |
| Runtime | Lambda (arm64) |
| Database | DynamoDB |
| CDN | CloudFront |
| Storage | S3 |
| Auth | OIDC (no long-lived credentials) |

### Directory Structure

```
infra/
├── bin/
│   └── app.ts                 # CDK app entry point
├── lib/
│   └── app-stack.ts           # Main stack definition
├── package.json
├── cdk.json
└── tsconfig.json

.github/
└── workflows/
    └── deploy.yml             # CI/CD pipeline
```

### CDK Patterns

#### Stack Structure

```typescript
import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';

export class AppStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);
    const env = this.node.tryGetContext('env') || 'dev';

    const table = new dynamodb.Table(this, 'Items', {
      tableName: `${env}-items`,
      partitionKey: { name: 'tenant_id', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'id', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      encryption: dynamodb.TableEncryption.AWS_MANAGED,
      pointInTimeRecovery: true,
    });

    const api = new lambda.Function(this, 'ApiHandler', {
      runtime: lambda.Runtime.PROVIDED_AL2,
      architecture: lambda.Architecture.ARM_64,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset('../backend/dist'),
      environment: { ENV: env, TABLE_NAME: table.tableName },
      timeout: cdk.Duration.seconds(30),
      memorySize: 256,
    });

    table.grantReadWriteData(api);
  }
}
```

### Environment Management

| Environment | Domain | Branch | Promotion |
|-------------|--------|--------|-----------|
| dev | `*.dev.digistratum.com` | `develop` | Auto on merge |
| staging | `*.staging.digistratum.com` | `release/*` | Auto on merge |
| prod | `*.digistratum.com` | `main` | Manual approval |

### Deployment

```bash
cd infra
npm ci
npx cdk synth -c env=dev      # Preview
npx cdk diff -c env=dev       # Show changes
npx cdk deploy -c env=dev     # Deploy
```

#### Lambda Hotfix

```bash
cd backend
GOOS=linux GOARCH=arm64 go build -o bootstrap ./cmd/lambda
zip -j /tmp/lambda.zip bootstrap
aws lambda update-function-code --function-name myapp-dev-api --zip-file fileb:///tmp/lambda.zip
```

### Monitoring Setup

```typescript
// CloudWatch Alarms
new cloudwatch.Alarm(this, 'ApiErrors', {
  metric: api.metricErrors({ period: cdk.Duration.minutes(5) }),
  threshold: 10,
  evaluationPeriods: 2,
});
```

### Infrastructure Common Mistakes

| ❌ Bad | ✅ Good |
|--------|---------|
| Hardcoded account/region | Use `cdk.Aws.ACCOUNT_ID` |
| Missing encryption | Always encrypt DynamoDB, S3 |
| Overly permissive IAM (`actions: ['*']`) | Use least privilege grants |
| Missing environment separation | Prefix all resources with `${env}-` |
| Long-lived credentials | Use OIDC for CI/CD |

---

## Security

### Quick Reference

| Requirement | Standard |
|-------------|----------|
| OWASP Top 10 | A01-A10 compliance |
| TLS | 1.2+ everywhere |
| Secrets | AWS Secrets Manager |
| Input validation | All endpoints |
| CORS | Allowed origins only |

### Authentication Model

DS apps use **SSO via DSAccount**. Apps never manage credentials directly.

```go
func RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session, err := validateSession(r)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        ctx := context.WithValue(r.Context(), SessionKey, session)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### OWASP Top 10 Checklist

| Control | Requirement |
|---------|-------------|
| A01: Access Control | Deny by default, tenant isolation, validate ownership |
| A02: Cryptographic | TLS 1.2+, HTTPS-only cookies, secrets in Secrets Manager |
| A03: Injection | Parameterized queries only, input validation, output encoding |
| A04: Insecure Design | Threat modeling, least privilege, defense in depth |
| A05: Misconfiguration | Security headers, no default credentials, generic error messages |
| A06: Vulnerable Components | Run `npm audit`, `govulncheck` regularly |
| A07: Auth Failures | Session timeout, secure token generation, rate limiting |
| A08: Data Integrity | Input validation, CSRF protection, webhook signatures |
| A09: Logging Failures | Log security events, no sensitive data in logs |
| A10: SSRF | URL validation, allowlist external calls |

### Input Validation

```go
func validate(input CreateItemRequest) error {
    if input.Title == "" {
        return errors.New("title is required")
    }
    if len(input.Title) > 200 {
        return errors.New("title too long (max 200)")
    }
    if !isValidTitle(input.Title) {
        return errors.New("title contains invalid characters")
    }
    return nil
}
```

| Field Type | Validation |
|------------|------------|
| String | Max length, character allowlist |
| Email | RFC 5322 pattern |
| URL | Protocol allowlist (https only) |
| ID | UUID format, ownership check |
| Numeric | Range bounds, type validation |

### Secrets Handling

```go
// ✅ Load from Secrets Manager
func getSecret(name string) (string, error) {
    svc := secretsmanager.New(sess)
    result, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{
        SecretId: aws.String(name),
    })
    return *result.SecretString, nil
}

// ❌ NEVER hardcode secrets or log them
```

### Security Common Mistakes

| ❌ Bad | ✅ Good |
|--------|---------|
| Exposing internal errors | Generic message, log details |
| Trusting client-provided tenant | Use validated session |
| Different errors for forbidden vs not found | Same error for both (no existence leak) |
| No audit trail | Log security-relevant actions |

---

## Testing

### Quick Targets

| Metric | Target | Gate |
|--------|--------|------|
| Backend unit coverage | ≥ 80% | CI |
| Frontend unit coverage | ≥ 70% | CI |
| E2E critical paths | 100% | CI |

### Test Pyramid

```
        ┌────────────┐
        │    E2E     │  10% - Full user flows
        ├────────────┤
        │ Integration│  20% - API + DB
        ├────────────┤
        │    Unit    │  70% - Isolated logic
        └────────────┘
```

### Test File Locations

Tests are co-located with source files:
- Backend: `items.go` → `items_test.go`
- Frontend: `Button.tsx` → `Button.test.tsx`
- E2E: `e2e/tests/*.spec.ts`

### Backend Test Pattern

```go
func TestGetItems(t *testing.T) {
    tests := []struct {
        name       string
        setup      func(*MockStorage)
        wantStatus int
        wantBody   string
    }{
        {
            name: "returns items when present",
            setup: func(m *MockStorage) {
                m.On("ListItems", mock.Anything).Return([]Item{{ID: "1"}}, nil)
            },
            wantStatus: http.StatusOK,
        },
        {
            name: "returns 500 on storage error",
            setup: func(m *MockStorage) {
                m.On("ListItems", mock.Anything).Return(nil, errors.New("db error"))
            },
            wantStatus: http.StatusInternalServerError,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mock := &MockStorage{}
            tt.setup(mock)
            handler := NewHandler(mock)
            req := httptest.NewRequest("GET", "/api/items", nil)
            w := httptest.NewRecorder()
            handler.GetItems(w, req)
            assert.Equal(t, tt.wantStatus, w.Code)
        })
    }
}
```

### Frontend Test Pattern

```tsx
describe('ItemList', () => {
  it('renders loading state', () => {
    render(<ItemList items={[]} isLoading={true} />);
    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it('renders items when present', () => {
    const items = [{ id: '1', name: 'Test Item' }];
    render(<ItemList items={items} isLoading={false} />);
    expect(screen.getByText('Test Item')).toBeInTheDocument();
  });
});
```

### E2E Test Pattern

```typescript
test('user can create and view item', async ({ page }) => {
  await page.click('[data-testid="create-item-btn"]');
  await page.fill('[name="title"]', 'Test Item');
  await page.click('[type="submit"]');
  await expect(page.getByText('Test Item')).toBeVisible();
});
```

### Running Tests

```bash
# Backend
cd backend && go test ./...
cd backend && go test -coverprofile=coverage.out ./...

# Frontend
cd frontend && pnpm test
cd frontend && pnpm test:coverage

# E2E
cd e2e && pnpm test
```

### When to Add Tests

| Change | Required Tests |
|--------|----------------|
| New endpoint | Unit + integration test |
| Bug fix | Regression test proving fix |
| New component | Unit test for behavior |
| New feature | E2E test for user flow |
| Refactor | Verify existing tests pass |

### Testing Common Mistakes

| ❌ Bad | ✅ Good |
|--------|---------|
| Testing implementation details | Test user-visible behavior |
| Brittle selectors (`div > div > button:nth-child(2)`) | Semantic selectors (`[data-testid]`, `getByRole`) |
| Only happy path tests | Test error paths too |
| Hardcoded test data | Generated/referenced data |
| Missing cleanup | Use `t.Cleanup()` or afterEach |

---

## See Also

- [reference/](../reference/) — Architecture, conventions, API standards
- [requirements/](../requirements/) — Detailed NFR specifications
- [runbooks/](../runbooks/) — Deployment and troubleshooting
