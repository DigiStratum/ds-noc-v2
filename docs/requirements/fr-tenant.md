# FR-TENANT: Multi-Tenant Support

> All DS ecosystem apps are multi-tenant by default.
> Data isolation is enforced at the data layer through partition keys.

---

## Requirements

### FR-TENANT-001: User session identifies current tenant

The session includes the currently active tenant context (or indicates personal/no-tenant mode).

**Acceptance Criteria:**
1. Session object contains `tenant_id` field (string, may be empty for personal context)
2. Tenant ID is set during SSO callback based on user's default/selected tenant
3. Personal context (no tenant) is explicitly supported with empty `tenant_id`
4. API responses include tenant context in response headers when applicable
5. Frontend can read current tenant from session without additional API calls

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/auth/session_test.go:TestTenantContext` |
| Unit test | `frontend/src/hooks/useAuth.test.tsx:returns tenant context` |
| Integration test | `backend/test/integration/tenant_test.go` |

**Evidence:**
- CI test results
- CloudWatch logs show `tenant_id` in structured request logs

---

### FR-TENANT-002: Users with multiple tenants can switch via nav dropdown

Users belonging to multiple tenants can switch between them without re-authenticating.

**Acceptance Criteria:**
1. Header shows current tenant name when user has tenant context
2. Dropdown lists all tenants the user belongs to
3. Selecting a different tenant updates session and refreshes data
4. Tenant switch preserves current page/route when possible
5. "Personal" option available for users with personal-context access
6. Single-tenant users see tenant name without dropdown

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `frontend/src/components/TenantSwitcher.test.tsx` |
| E2E test | `frontend/e2e/navigation.spec.ts:tenant switching` |

**Evidence:**
- CI test results
- Screenshot showing tenant dropdown with multiple options

---

### FR-TENANT-003: All data queries are scoped to current tenant

Database queries automatically filter to the current tenant, preventing cross-tenant data access.

**Acceptance Criteria:**
1. All DynamoDB queries include tenant_id in partition key condition
2. Repository layer extracts tenant_id from context, not from request parameters
3. Queries without tenant context fail safely (empty results or error)
4. No query patterns allow scanning across tenant boundaries
5. Test coverage includes cross-tenant access denial scenarios

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/dynamo/repository_test.go:TestTenantScoping` |
| Unit test | `backend/internal/dynamo/repository_test.go:TestCrossTenantDenied` |
| Integration test | `backend/test/integration/tenant_isolation_test.go` |

**Evidence:**
- CI test results
- Code review checklist includes tenant scoping verification

---

### FR-TENANT-004: API requests include X-Tenant-ID header

Backend populates tenant context in response headers for debugging and client awareness.

**Acceptance Criteria:**
1. `X-Tenant-ID` header included in all authenticated API responses
2. Header value matches session tenant context
3. Header omitted for unauthenticated requests
4. Frontend logs include tenant context for debugging
5. Header is informational only — tenant is NOT derived from request header

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/middleware/tenant_test.go:TestTenantHeader` |
| E2E test | `frontend/e2e/api-integration.spec.ts:includes tenant header` |

**Evidence:**
- CI test results
- `curl -v` shows `X-Tenant-ID` in response headers

---

## Implementation

### Backend Middleware

```go
// backend/internal/middleware/tenant.go
func TenantHeader(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session := auth.GetSession(r.Context())
        if session != nil && session.TenantID != "" {
            w.Header().Set("X-Tenant-ID", session.TenantID)
        }
        next.ServeHTTP(w, r)
    })
}
```

### Repository Pattern

```go
// backend/internal/dynamo/repository.go
func (r *Repository) GetItems(ctx context.Context) ([]Item, error) {
    session := auth.GetSession(ctx)
    if session == nil {
        return nil, ErrNoSession
    }
    
    // Tenant ID in partition key ensures isolation
    pk := fmt.Sprintf("TENANT#%s", session.TenantID)
    
    input := &dynamodb.QueryInput{
        TableName:              aws.String(r.tableName),
        KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":pk": &types.AttributeValueMemberS{Value: pk},
            ":sk": &types.AttributeValueMemberS{Value: "ITEM#"},
        },
    }
    
    return r.query(ctx, input)
}
```

### Frontend Hook

```tsx
// frontend/src/hooks/useTenant.tsx
export function useTenant() {
  const { data: session } = useAuth();
  const queryClient = useQueryClient();
  
  const switchTenant = async (tenantId: string) => {
    await fetch('/api/auth/switch-tenant', {
      method: 'POST',
      body: JSON.stringify({ tenant_id: tenantId }),
    });
    queryClient.invalidateQueries(); // Refresh all data
  };
  
  return {
    currentTenant: session?.tenant,
    tenants: session?.available_tenants ?? [],
    switchTenant,
  };
}
```

---

## Security Considerations

- Tenant ID MUST come from validated session, never from request parameters
- Cross-tenant access attempts should be logged as security events
- Test suite must include explicit cross-tenant denial tests
- Admin/superuser access patterns must be explicitly designed and audited

---

## Traceability

| Requirement | Implementation | Test | Status |
|-------------|----------------|------|--------|
| FR-TENANT-001 | `backend/internal/auth/session.go` | `session_test.go`, `useAuth.test.tsx` | ⚠️ |
| FR-TENANT-002 | `frontend/src/components/TenantSwitcher.tsx` | `TenantSwitcher.test.tsx`, `navigation.spec.ts` | ⚠️ |
| FR-TENANT-003 | Repository layer tenant scoping | `repository_test.go`, `tenant_isolation_test.go` | ⚠️ |
| FR-TENANT-004 | `backend/internal/middleware/tenant.go` | `tenant_test.go`, `api-integration.spec.ts` | ⚠️ |

---

*Last updated: 2026-03-23*
