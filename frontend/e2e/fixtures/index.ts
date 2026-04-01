/**
 * Playwright Test Fixtures for DS Apps
 *
 * Provides:
 * - authenticatedPage: Page with ds_session cookie set (DS SSO pattern)
 * - apiClient: Authenticated HTTP client for direct API calls
 *
 * DS SSO Pattern:
 * - DSAccount owns all sessions (account.digistratum.com)
 * - Apps read ds_session cookie from .digistratum.com domain
 * - Apps validate via DSAccount /api/auth/me endpoint
 *
 * Environment Variables:
 * - DS_SESSION: Valid ds_session cookie value (required for auth tests)
 * - APP_URL: App base URL (defaults to localhost:5173)
 * - API_URL: Backend API URL (defaults to localhost:3001)
 *
 * @see https://playwright.dev/docs/test-fixtures
 */

import { test as base, Page, BrowserContext, APIRequestContext } from '@playwright/test';

// Environment configuration
const APP_URL = process.env.APP_URL || 'http://localhost:5173';
const API_URL = process.env.API_URL || 'http://localhost:3001';
const DS_SESSION = process.env.DS_SESSION;
const COOKIE_DOMAIN = process.env.SSO_COOKIE_DOMAIN || '.digistratum.com';

/**
 * Authenticated API client for direct backend calls
 */
export interface ApiClient {
  get: (path: string) => Promise<Response>;
  post: (path: string, body?: unknown) => Promise<Response>;
  put: (path: string, body?: unknown) => Promise<Response>;
  patch: (path: string, body?: unknown) => Promise<Response>;
  delete: (path: string) => Promise<Response>;
  baseUrl: string;
}

/**
 * Test fixtures type definition
 */
type TestFixtures = {
  /** Page with ds_session cookie pre-set for authenticated tests */
  authenticatedPage: Page;
  /** API client with ds_session cookie for direct API calls */
  apiClient: ApiClient;
  /** Raw ds_session cookie value (from env) */
  sessionCookie: string | undefined;
};

/**
 * Create an API client with ds_session cookie authentication
 */
function createApiClient(sessionCookie?: string): ApiClient {
  const makeRequest = async (
    method: string,
    path: string,
    body?: unknown
  ): Promise<Response> => {
    const url = path.startsWith('http') ? path : `${API_URL}${path}`;
    const headers: Record<string, string> = {};

    // DS apps use ds_session cookie for auth
    if (sessionCookie) {
      headers['Cookie'] = `ds_session=${sessionCookie}`;
    }

    if (body) {
      headers['Content-Type'] = 'application/json';
    }

    return fetch(url, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
      credentials: 'include',
    });
  };

  return {
    get: (path: string) => makeRequest('GET', path),
    post: (path: string, body?: unknown) => makeRequest('POST', path, body),
    put: (path: string, body?: unknown) => makeRequest('PUT', path, body),
    patch: (path: string, body?: unknown) => makeRequest('PATCH', path, body),
    delete: (path: string) => makeRequest('DELETE', path),
    baseUrl: API_URL,
  };
}

/**
 * Extended Playwright test with DS-specific fixtures
 */
export const test = base.extend<TestFixtures>({
  /**
   * Session cookie from environment
   */
  sessionCookie: [DS_SESSION, { option: true }],

  /**
   * Page fixture with ds_session cookie pre-set
   *
   * Usage:
   * ```ts
   * test('authenticated user can access dashboard', async ({ authenticatedPage }) => {
   *   await authenticatedPage.goto('/dashboard');
   *   await expect(authenticatedPage).toHaveURL(/dashboard/);
   * });
   * ```
   */
  authenticatedPage: async ({ browser, sessionCookie }, use) => {
    if (!sessionCookie) {
      throw new Error(
        'DS_SESSION environment variable required for authenticated tests. ' +
        'Get a valid session cookie from DSAccount.'
      );
    }

    // Create a new context with the session cookie
    const context = await browser.newContext();

    // Set ds_session cookie
    // For local dev, we use localhost; for deployed envs, use the actual domain
    const isLocalhost = APP_URL.includes('localhost') || APP_URL.includes('127.0.0.1');
    const cookieDomain = isLocalhost ? 'localhost' : COOKIE_DOMAIN;

    await context.addCookies([
      {
        name: 'ds_session',
        value: sessionCookie,
        domain: cookieDomain,
        path: '/',
        httpOnly: true,
        secure: !isLocalhost,
        sameSite: 'Lax',
      },
    ]);

    const page = await context.newPage();

    await use(page);

    // Cleanup
    await context.close();
  },

  /**
   * API client fixture for direct backend calls
   *
   * Usage:
   * ```ts
   * test('can fetch items via API', async ({ apiClient }) => {
   *   const response = await apiClient.get('/api/items');
   *   expect(response.status).toBe(200);
   * });
   * ```
   */
  apiClient: async ({ sessionCookie }, use) => {
    const client = createApiClient(sessionCookie);
    await use(client);
  },
});

/**
 * Re-export expect for convenience
 */
export { expect } from '@playwright/test';

/**
 * Helper to create an unauthenticated API client (for 401 tests)
 */
export function unauthenticatedApiClient(): ApiClient {
  return createApiClient(undefined);
}

/**
 * Helper to get a fresh unauthenticated page
 */
export async function getUnauthenticatedPage(browser: BrowserContext): Promise<Page> {
  return browser.newPage();
}
