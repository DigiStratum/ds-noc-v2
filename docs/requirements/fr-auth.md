# FR-AUTH: Authentication & Authorization

> Standard DigiStratum SSO authentication via DSAccount.
> All DS ecosystem apps delegate authentication to DSAccount — no custom password storage.

---

## Requirements

### FR-AUTH-001: Users authenticate via DSAccount SSO

Users access the application through centralized SSO authentication provided by DSAccount.

**Acceptance Criteria:**
1. Unauthenticated user visiting protected route receives 302 redirect to DSAccount login
2. Redirect URL includes `?redirect=` parameter with original path (URL-encoded)
3. After SSO success, user returns to original path with valid session cookie
4. Session cookie: `ds_session`, HttpOnly, Secure (in prod), SameSite=Lax, domain=`.digistratum.com`
5. Session is validated server-side via DSAccount `/api/auth/me` endpoint

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/auth/middleware_test.go:TestRequireAuth` |
| E2E test | `frontend/e2e/auth.spec.ts` |
| Integration test | `backend/test/integration/auth_test.go` |

**Evidence:**
- CI test results (GitHub Actions)
- Manual verification: Incognito browser → protected route → observe redirect

---

### FR-AUTH-002: Unauthenticated requests redirect to SSO login

API and page requests without valid session are redirected (web) or rejected (API).

**Acceptance Criteria:**
1. Web requests to protected routes without `ds_session` cookie → 302 redirect to DSAccount login
2. API requests to protected endpoints without valid session → 401 Unauthorized (JSON response)
3. Redirect preserves original requested path for post-login return
4. Public routes (`/health`, `/api/health`, `/api/build`) remain accessible without auth

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/auth/middleware_test.go:TestRequireAuth_NoSession` |
| Unit test | `backend/internal/auth/middleware_test.go:TestPublicRoutes` |
| E2E test | `frontend/e2e/auth.spec.ts:redirects to login` |

**Evidence:**
- CI test results
- `curl -v https://app.digistratum.com/api/items` returns 401

---

### FR-AUTH-003: Session includes user identity and tenant context

Authenticated sessions contain user and tenant information for authorization decisions.

**Acceptance Criteria:**
1. Session context includes: `user_id`, `email`, `name`, `tenant_id` (or empty for personal)
2. Session data available in Go context via `auth.GetSession(ctx)`
3. Session data available in React via `useAuth()` hook
4. Tenant ID extracted from session, not URL or request parameter
5. Invalid or expired session returns 401 (not stale data)

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/auth/session_test.go:TestSessionContext` |
| Unit test | `frontend/src/hooks/useAuth.test.tsx` |
| Integration test | `backend/test/integration/auth_test.go:TestSessionData` |

**Evidence:**
- CI test results
- CloudWatch logs showing user_id/tenant_id in structured logs

---

### FR-AUTH-004: Logout clears session and redirects to DSAccount logout

Logout terminates the session and redirects to centralized DSAccount logout.

**Acceptance Criteria:**
1. Logout action clears local session state
2. User is redirected to `https://account.digistratum.com/logout?redirect={app_url}`
3. After DSAccount logout, user returns to app in unauthenticated state
4. Subsequent requests require fresh authentication
5. Session cookie is invalidated/expired

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `backend/internal/auth/handlers_test.go:TestLogoutHandler` |
| E2E test | `frontend/e2e/auth.spec.ts:logout flow` |

**Evidence:**
- CI test results
- Manual verification: login → logout → verify redirect and session cleared

---

## Implementation

### Backend

```go
// backend/internal/auth/middleware.go
func RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session, err := validateSession(r)
        if err != nil {
            if isAPIRequest(r) {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
            } else {
                redirectURL := buildSSORedirect(r)
                http.Redirect(w, r, redirectURL, http.StatusFound)
            }
            return
        }
        ctx := context.WithValue(r.Context(), SessionKey, session)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Frontend

```tsx
// frontend/src/hooks/useAuth.tsx
export function useAuth() {
  const { data: session, isLoading, error } = useQuery({
    queryKey: ['session'],
    queryFn: () => fetch('/api/auth/me').then(r => r.json()),
  });
  
  return {
    user: session?.user,
    tenantId: session?.tenant_id,
    isAuthenticated: !!session?.user,
    isLoading,
  };
}
```

---

## Traceability

| Requirement | Implementation | Test | Status |
|-------------|----------------|------|--------|
| FR-AUTH-001 | `backend/internal/auth/middleware.go` | `middleware_test.go`, `auth.spec.ts` | ⚠️ |
| FR-AUTH-002 | `backend/internal/auth/middleware.go:RequireAuth` | `middleware_test.go` | ⚠️ |
| FR-AUTH-003 | `backend/internal/auth/session.go`, `frontend/src/hooks/useAuth.tsx` | `session_test.go`, `useAuth.test.tsx` | ⚠️ |
| FR-AUTH-004 | `backend/internal/auth/handlers.go:LogoutHandler` | `handlers_test.go`, `auth.spec.ts` | ⚠️ |

---

*Last updated: 2026-03-23*
