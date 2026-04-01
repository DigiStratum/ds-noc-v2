/**
 * Production Smoke Tests
 *
 * Critical path tests run after prod deployment.
 * Tag: @smoke - These tests are filtered by smoke-prod.yml workflow
 *
 * These tests should:
 * - Complete quickly (<30s total)
 * - Test only critical paths
 * - Have minimal dependencies
 * - Be highly reliable (no flakes)
 *
 * @see .github/workflows/smoke-prod.yml
 */

import {
  createTestUser,
  authenticateAs,
  cleanupTestUser,
  unauthenticatedClient,
  TestUser,
  AuthenticatedClient,
} from '../fixtures/test-user';

describe('@smoke Production Smoke Tests', () => {
  describe('Health Check', () => {
    /**
     * @smoke
     * @covers NFR-100 Service availability
     *
     * Most critical test - if this fails, service is down.
     */
    it('@smoke health endpoint returns 200', async () => {
      const client = unauthenticatedClient();
      const response = await client.get('/api/health');

      expect(response.status).toBe(200);
      const data = await response.json();
      expect(data.status).toBe('healthy');
    });
  });

  describe('Authentication Flow', () => {
    let testUser: TestUser;
    let client: AuthenticatedClient;

    beforeAll(async () => {
      testUser = await createTestUser();
    });

    afterAll(async () => {
      await cleanupTestUser(testUser);
    });

    /**
     * @smoke
     * @covers SEC-001 SSO authentication flow
     *
     * Verifies auth flow is working end-to-end.
     */
    it('@smoke authenticated user can access protected endpoints', async () => {
      client = await authenticateAs(testUser);
      const response = await client.get('/api/me');

      expect(response.status).toBe(200);
      const data = await response.json();
      expect(data.user).toBeDefined();
      expect(data.user.email).toBe(testUser.email);
    });

    /**
     * @smoke
     * @covers SEC-001 Authentication required for API access
     */
    it('@smoke unauthenticated requests are rejected', async () => {
      const unauthed = unauthenticatedClient();
      const response = await unauthed.get('/api/me');

      expect(response.status).toBe(401);
    });
  });

  describe('Core API Endpoints', () => {
    let client: AuthenticatedClient;
    let testUser: TestUser;

    beforeAll(async () => {
      testUser = await createTestUser();
      client = await authenticateAs(testUser);
    });

    afterAll(async () => {
      await cleanupTestUser(testUser);
    });

    /**
     * @smoke
     * @covers FR-001 List items API availability
     *
     * Verifies core list endpoint responds correctly.
     */
    it('@smoke list items returns 200', async () => {
      const response = await client.get('/api/items');

      expect(response.status).toBe(200);
      const data = await response.json();
      expect(Array.isArray(data.items)).toBe(true);
    });
  });
});

describe('@smoke Frontend Smoke Tests', () => {
  /**
   * @smoke
   * @covers NFR-101 Frontend asset delivery
   *
   * Verifies CloudFront is serving frontend assets.
   */
  it('@smoke homepage loads', async () => {
    const client = unauthenticatedClient();
    const baseUrl = process.env.APP_URL || 'http://localhost:5173';

    const response = await fetch(baseUrl);
    expect(response.status).toBe(200);

    const html = await response.text();
    expect(html).toContain('<!DOCTYPE html>');
    expect(html).toContain('<div id="root">');
  });

  /**
   * @smoke
   * @covers NFR-101 Frontend asset delivery
   *
   * Verifies JS bundle is accessible (critical for SPA).
   */
  it('@smoke JS bundle is accessible', async () => {
    const client = unauthenticatedClient();
    const baseUrl = process.env.APP_URL || 'http://localhost:5173';

    // Fetch index to find JS bundle path
    const indexResponse = await fetch(baseUrl);
    const html = await indexResponse.text();

    // Extract first JS bundle reference
    const jsMatch = html.match(/src="(\/assets\/index-[^"]+\.js)"/);
    if (jsMatch) {
      const jsUrl = `${baseUrl}${jsMatch[1]}`;
      const jsResponse = await fetch(jsUrl);
      expect(jsResponse.status).toBe(200);
      expect(jsResponse.headers.get('content-type')).toContain('javascript');
    }
  });
});
