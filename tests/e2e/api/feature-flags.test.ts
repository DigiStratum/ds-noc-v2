/**
 * FeatureFlags API E2E Tests
 *
 * Tests for feature flags CRUD operations.
 * @covers FR-API-FF Feature Flags API
 */

import {
  createTestUser,
  authenticateAs,
  cleanupTestUser,
  unauthenticatedClient,
  TestUser,
  AuthenticatedClient,
} from '../fixtures/test-user';

describe('Feature Flags API', () => {
  let testUser: TestUser;
  let client: AuthenticatedClient;

  beforeAll(async () => {
    testUser = await createTestUser();
    client = await authenticateAs(testUser);
  });

  afterAll(async () => {
    await cleanupTestUser(testUser);
  });

  describe('GET /api/feature-flags', () => {
    /**
     * @covers FR-API-FF-LIST List all feature flags
     */
    it('returns 200 with list of flags', async () => {
      const response = await client.get('/api/feature-flags');

      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('_links');
      expect(response.body).toHaveProperty('flags');
      expect(response.body).toHaveProperty('count');
    });
  });

  describe('GET /api/feature-flags/evaluate', () => {
    /**
     * @covers FR-API-FF-EVAL Evaluate flags for current context
     */
    it('returns 200 with evaluated flags map', async () => {
      const response = await client.get('/api/feature-flags/evaluate');

      expect(response.status).toBe(200);
      expect(response.body).toHaveProperty('_links');
      expect(response.body).toHaveProperty('flags');
      // flags should be a map of string -> boolean
      if (Object.keys(response.body.flags).length > 0) {
        const firstKey = Object.keys(response.body.flags)[0];
        expect(typeof response.body.flags[firstKey]).toBe('boolean');
      }
    });
  });

  describe('PATCH /api/feature-flags/{key}', () => {
    const testFlagKey = 'e2e-test-flag';

    afterAll(async () => {
      // Cleanup test flag
      await client.delete(`/api/feature-flags/${testFlagKey}`);
    });

    /**
     * @covers FR-API-FF-UPDATE Create/update feature flag
     */
    it('creates a new flag when it does not exist', async () => {
      const response = await client.patch(`/api/feature-flags/${testFlagKey}`, {
        enabled: true,
        description: 'E2E test flag',
        percentage: 50,
      });

      expect(response.status).toBe(200);
      expect(response.body.key).toBe(testFlagKey);
      expect(response.body.enabled).toBe(true);
      expect(response.body.percentage).toBe(50);
    });

    /**
     * @covers FR-API-FF-UPDATE Update existing flag
     */
    it('updates an existing flag', async () => {
      const response = await client.patch(`/api/feature-flags/${testFlagKey}`, {
        enabled: false,
        percentage: 25,
      });

      expect(response.status).toBe(200);
      expect(response.body.enabled).toBe(false);
      expect(response.body.percentage).toBe(25);
    });
  });

  describe('DELETE /api/feature-flags/{key}', () => {
    const testFlagKey = 'e2e-test-flag-delete';

    beforeEach(async () => {
      // Create a flag to delete
      await client.patch(`/api/feature-flags/${testFlagKey}`, {
        enabled: true,
        description: 'Flag to be deleted',
      });
    });

    /**
     * @covers FR-API-FF-DELETE Delete a feature flag
     */
    it('returns 204 on successful deletion', async () => {
      const response = await client.delete(`/api/feature-flags/${testFlagKey}`);

      expect(response.status).toBe(204);
    });
  });
});
