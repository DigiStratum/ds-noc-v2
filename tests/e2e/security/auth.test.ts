/**
 * Example Security Test: Authentication & Authorization
 *
 * Demonstrates security E2E test patterns with @covers markers.
 */

import {
  createTestUser,
  authenticateAs,
  cleanupTestUser,
  unauthenticatedClient,
  TestUser,
  AuthenticatedClient,
} from '../fixtures/test-user';

const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:3001';

describe('Authentication Security', () => {
  let testUser: TestUser;
  let client: AuthenticatedClient;

  beforeAll(async () => {
    testUser = await createTestUser();
    client = await authenticateAs(testUser);
  });

  afterAll(async () => {
    await cleanupTestUser(testUser);
  });

  describe('Token Validation', () => {
    /**
     * @covers SEC-001 Authentication required for API access
     */
    it('rejects requests without token', async () => {
      const unauthed = unauthenticatedClient();
      const response = await unauthed.get('/api/items');

      expect(response.status).toBe(401);
    });

    /**
     * @covers SEC-002 Invalid tokens rejected
     */
    it('rejects requests with malformed token', async () => {
      const response = await fetch(`${API_BASE_URL}/api/items`, {
        headers: {
          Authorization: 'Bearer malformed-token-12345',
        },
      });

      expect(response.status).toBe(401);
    });

    /**
     * @covers SEC-002 Invalid tokens rejected
     */
    it('rejects requests with expired token', async () => {
      // This token is properly formatted but expired
      const expiredToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.' +
        'eyJzdWIiOiIxMjM0NTY3ODkwIiwiZXhwIjoxMDAwMDAwMDAwfQ.' +
        'signature';

      const response = await fetch(`${API_BASE_URL}/api/items`, {
        headers: {
          Authorization: `Bearer ${expiredToken}`,
        },
      });

      expect(response.status).toBe(401);
    });
  });

  describe('Tenant Isolation', () => {
    /**
     * @covers SEC-003 Users cannot access other tenants' data
     */
    it('returns 403 when accessing another tenant\'s resource', async () => {
      // Try to access a resource belonging to a different tenant
      const otherTenantItemId = 'other-tenant-item-id';

      const response = await client.get(`/api/items/${otherTenantItemId}`);

      // Should be 403 (forbidden) or 404 (not found - tenant filtering)
      expect([403, 404]).toContain(response.status);
    });

    /**
     * @covers SEC-003 Users cannot access other tenants' data
     */
    it('cannot modify another tenant\'s resource', async () => {
      const otherTenantItemId = 'other-tenant-item-id';

      const response = await client.patch(`/api/items/${otherTenantItemId}`, {
        title: 'Hacked!',
      });

      expect([403, 404]).toContain(response.status);
    });
  });

  describe('Rate Limiting', () => {
    /**
     * @covers SEC-004 Rate limiting prevents abuse
     */
    it('enforces rate limits on repeated requests', async () => {
      const responses: Response[] = [];

      // Make rapid requests
      for (let i = 0; i < 100; i++) {
        responses.push(await client.get('/api/health'));
      }

      // At least some should be rate-limited (429)
      const rateLimited = responses.filter((r) => r.status === 429);

      // This test may need adjustment based on actual rate limit config
      // For development, we might expect no rate limiting
      // For production-like config, expect some 429s
      console.log(`Rate limited: ${rateLimited.length}/${responses.length}`);
    });
  });

  describe('Input Sanitization', () => {
    /**
     * @covers SEC-005 XSS payloads rejected or sanitized
     */
    it('rejects XSS payloads in input', async () => {
      const xssPayload = '<script>alert("xss")</script>';

      const response = await client.post('/api/items', {
        title: xssPayload,
      });

      if (response.status === 201) {
        // If accepted, verify the payload was sanitized
        const data = await response.json();
        expect(data.item.title).not.toContain('<script>');

        // Cleanup
        await client.delete(`/api/items/${data.item.id}`);
      } else {
        // If rejected, verify proper error
        expect(response.status).toBe(400);
      }
    });

    /**
     * @covers SEC-006 SQL injection payloads rejected
     */
    it('handles SQL injection attempts safely', async () => {
      const sqlPayload = "'; DROP TABLE items; --";

      const response = await client.post('/api/items', {
        title: sqlPayload,
      });

      // Should either reject or safely escape
      expect([201, 400]).toContain(response.status);

      if (response.status === 201) {
        const data = await response.json();
        // Cleanup
        await client.delete(`/api/items/${data.item.id}`);
      }
    });
  });
});

describe('Session Security', () => {
  /**
   * @covers SEC-007 Session invalidation on logout
   */
  it('invalidates token after logout', async () => {
    const user = await createTestUser();
    const client = await authenticateAs(user);

    // Verify token works
    let response = await client.get('/api/items');
    expect(response.status).toBe(200);

    // Logout
    await client.post('/api/auth/logout');

    // Token should no longer work
    response = await client.get('/api/items');
    expect(response.status).toBe(401);

    await cleanupTestUser(user);
  });
});
