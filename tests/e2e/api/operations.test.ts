/**
 * Operations API E2E Tests
 *
 * Auto-generated test stubs for GET /api/operations
 * Update @covers markers when requirements are assigned.
 */

import {
  createTestUser,
  authenticateAs,
  cleanupTestUser,
  unauthenticatedClient,
  TestUser,
  AuthenticatedClient,
} from '../fixtures/test-user';

describe('ListOperations', () => {
  let testUser: TestUser;
  let client: AuthenticatedClient;

  beforeAll(async () => {
    testUser = await createTestUser();
    client = await authenticateAs(testUser);
  });

  afterAll(async () => {
    await cleanupTestUser(testUser);
  });

  describe('GET /api/operations', () => {
    /**
     * @covers FR-API-XXX TODO: Assign requirement ID
     */
    it('returns 200 for authenticated request', async () => {
      const response = await client.get('/api/operations');

      expect(response.status).toBe(200);
      // TODO: Add response body assertions
    });

    /**
     * @covers FR-SEC-XXX Authentication required
     */
    it('returns 401 for unauthenticated request', async () => {
      const unauthed = unauthenticatedClient();
      const response = await unauthed.get('/api/operations');

      expect(response.status).toBe(401);
    });
  });
});
