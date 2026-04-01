/**
 * Example API Test: Items CRUD
 *
 * Demonstrates API E2E test patterns with @covers markers.
 */

import {
  createTestUser,
  authenticateAs,
  cleanupTestUser,
  unauthenticatedClient,
  TestUser,
  AuthenticatedClient,
} from '../fixtures/test-user';

describe('Items API', () => {
  let testUser: TestUser;
  let client: AuthenticatedClient;
  let createdItemId: string;

  beforeAll(async () => {
    testUser = await createTestUser();
    client = await authenticateAs(testUser);
  });

  afterAll(async () => {
    // Cleanup created items
    if (createdItemId) {
      await client.delete(`/api/items/${createdItemId}`);
    }
    await cleanupTestUser(testUser);
  });

  describe('GET /api/items', () => {
    /**
     * @covers FR-001 List items for current tenant
     */
    it('returns 200 with items for authenticated user', async () => {
      const response = await client.get('/api/items');

      expect(response.status).toBe(200);
      const data = await response.json();
      expect(Array.isArray(data.items)).toBe(true);
    });

    /**
     * @covers SEC-001 Authentication required for API access
     */
    it('returns 401 for unauthenticated request', async () => {
      const unauthed = unauthenticatedClient();
      const response = await unauthed.get('/api/items');

      expect(response.status).toBe(401);
    });
  });

  describe('POST /api/items', () => {
    /**
     * @covers FR-002 Create item with title and description
     */
    it('creates item and returns 201', async () => {
      const newItem = {
        title: 'Test Item',
        description: 'Created by E2E test',
      };

      const response = await client.post('/api/items', newItem);

      expect(response.status).toBe(201);
      const data = await response.json();
      expect(data.item.title).toBe(newItem.title);
      expect(data.item.id).toBeDefined();

      createdItemId = data.item.id;
    });

    /**
     * @covers FR-002 Create item with title and description
     * @covers NFR-001 Input validation on all endpoints
     */
    it('returns 400 for missing required fields', async () => {
      const response = await client.post('/api/items', {});

      expect(response.status).toBe(400);
      const data = await response.json();
      expect(data.error).toBeDefined();
    });

    /**
     * @covers SEC-001 Authentication required for API access
     */
    it('returns 401 for unauthenticated request', async () => {
      const unauthed = unauthenticatedClient();
      const response = await unauthed.post('/api/items', { title: 'Test' });

      expect(response.status).toBe(401);
    });
  });

  describe('GET /api/items/:id', () => {
    /**
     * @covers FR-003 Get item by ID
     */
    it('returns 200 with item details', async () => {
      // Ensure item exists
      if (!createdItemId) {
        const createResponse = await client.post('/api/items', { title: 'Get Test' });
        createdItemId = (await createResponse.json()).item.id;
      }

      const response = await client.get(`/api/items/${createdItemId}`);

      expect(response.status).toBe(200);
      const data = await response.json();
      expect(data.item.id).toBe(createdItemId);
    });

    /**
     * @covers FR-003 Get item by ID
     */
    it('returns 404 for non-existent item', async () => {
      const response = await client.get('/api/items/non-existent-id-12345');

      expect(response.status).toBe(404);
    });
  });

  describe('DELETE /api/items/:id', () => {
    /**
     * @covers FR-004 Delete item
     */
    it('returns 204 on successful deletion', async () => {
      // Create item to delete
      const createResponse = await client.post('/api/items', { title: 'Delete Me' });
      const itemToDelete = (await createResponse.json()).item.id;

      const response = await client.delete(`/api/items/${itemToDelete}`);

      expect(response.status).toBe(204);

      // Verify deleted
      const getResponse = await client.get(`/api/items/${itemToDelete}`);
      expect(getResponse.status).toBe(404);
    });

    /**
     * @covers FR-004 Delete item
     */
    it('returns 404 for non-existent item', async () => {
      const response = await client.delete('/api/items/non-existent-id-12345');

      expect(response.status).toBe(404);
    });
  });
});
