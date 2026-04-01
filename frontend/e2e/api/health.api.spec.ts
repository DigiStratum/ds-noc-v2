/**
 * Health Check API Tests
 *
 * Example API test using DS fixtures.
 * Tests against the health endpoint (no auth required).
 */

import { test, expect, unauthenticatedApiClient } from '../fixtures';

test.describe('Health API', () => {
  /**
   * @covers NFR-OPS-001 Health check endpoint available
   */
  test('GET /api/health returns 200', async () => {
    const client = unauthenticatedApiClient();
    const response = await client.get('/api/health');

    expect(response.status).toBe(200);
    const data = await response.json();
    expect(data.status).toBe('ok');
  });
});

test.describe('Authenticated API', () => {
  /**
   * @covers SEC-001 API requires authentication
   */
  test('unauthenticated request returns 401', async () => {
    const client = unauthenticatedApiClient();
    const response = await client.get('/api/items');

    expect(response.status).toBe(401);
  });

  /**
   * @covers FR-001 Authenticated users can list items
   */
  test('authenticated request returns 200', async ({ apiClient }) => {
    const response = await apiClient.get('/api/items');

    expect(response.status).toBe(200);
    const data = await response.json();
    expect(Array.isArray(data.items || data)).toBe(true);
  });
});
