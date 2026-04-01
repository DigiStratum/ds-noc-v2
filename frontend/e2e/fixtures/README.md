# E2E Test Fixtures

Playwright test fixtures for DS apps with ds_session cookie authentication.

## Fixtures

| Fixture | Purpose |
|---------|---------|
| `authenticatedPage` | Page with ds_session cookie pre-set |
| `apiClient` | HTTP client with ds_session cookie for API calls |
| `sessionCookie` | Raw ds_session value from environment |

## Usage

```typescript
import { test, expect, unauthenticatedApiClient } from './fixtures';

// Authenticated UI test
test('dashboard loads for authenticated user', async ({ authenticatedPage }) => {
  await authenticatedPage.goto('/dashboard');
  await expect(authenticatedPage).not.toHaveURL(/login/);
});

// Authenticated API test
test('can fetch items', async ({ apiClient }) => {
  const response = await apiClient.get('/api/items');
  expect(response.status).toBe(200);
});

// Unauthenticated test (401 scenarios)
test('unauthenticated requests are rejected', async () => {
  const client = unauthenticatedApiClient();
  const response = await client.get('/api/items');
  expect(response.status).toBe(401);
});
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DS_SESSION` | For auth tests | Valid ds_session cookie value |
| `APP_URL` | No | Frontend URL (default: localhost:5173) |
| `API_URL` | No | Backend API URL (default: localhost:3001) |
| `SSO_COOKIE_DOMAIN` | No | Cookie domain (default: .digistratum.com) |

## Getting a Test Session

For local development, get a session from DSAccount:

1. Log in to account.digistratum.com
2. Open DevTools → Application → Cookies
3. Copy the `ds_session` value
4. Export: `export DS_SESSION="<value>"`

For CI, use a dedicated test user with a long-lived session or service token.

## DS SSO Pattern

DS apps don't manage their own sessions. DSAccount is the central auth provider:

- DSAccount sets `ds_session` cookie on `.digistratum.com`
- Apps read this cookie and validate via `/api/auth/me`
- Apps never set or clear the session cookie
- Login/logout redirects to DSAccount

The fixtures mirror this by injecting the ds_session cookie into the browser context.
