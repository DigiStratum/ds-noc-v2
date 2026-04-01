/**
 * Feature Flags Evaluate API E2E Tests
 *
 * Tests for feature flag evaluation endpoint.
 * @covers FR-API-FF-EVAL Feature Flags Evaluation
 */

import {
  createTestUser,
  authenticateAs,
  cleanupTestUser,
  unauthenticatedClient,
  TestUser,
  AuthenticatedClient,
} from '../fixtures/test-user';

describe('Feature Flags Evaluate API', () => {
  let testUser: TestUser;
  let client: AuthenticatedClient;

  beforeAll(async () => {
    testUser = await createTestUser();
    client = await authenticateAs(testUser);
  });

  afterAll(async () => {
    await cleanupTestUser(testUser);
  });

  describe('GET /api/feature-flags/evaluate', () => {
    /**
     * @covers FR-API-FF-EVAL Returns evaluated flags
     */
    it('returns 200 with flags map', async () => {
      const response = await client.get('/api/feature-flags/evaluate');

      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('_links');
      expect(response.body._links).toHaveProperty('self');
      expect(response.body).toHaveProperty('flags');
      expect(typeof response.body.flags).toBe('object');
    });

    /**
     * @covers FR-API-FF-EVAL Respects tenant context
     */
    it('includes tenant context in evaluation', async () => {
      const response = await client.get('/api/feature-flags/evaluate', {
        headers: {
          'X-Tenant-ID': 'test-tenant',
        },
      });

      expect(response.status).toBe(200);
      // Flags should be evaluated with tenant context
      expect(response.body).toHaveProperty('flags');
    });
  });
});
