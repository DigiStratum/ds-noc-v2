# Test Fixtures

Shared test data and helper utilities for E2E tests.

## Files

| File | Purpose |
|------|---------|
| `test-user.ts` | Test user creation and authentication helpers |

## Usage

```typescript
import { createTestUser, authenticateAs, cleanupTestUser } from '../fixtures/test-user';

describe('Item API', () => {
  let testUser: TestUser;

  beforeAll(async () => {
    testUser = await createTestUser();
  });

  afterAll(async () => {
    await cleanupTestUser(testUser);
  });

  it('creates item as authenticated user', async () => {
    const client = await authenticateAs(testUser);
    const response = await client.post('/api/items', { title: 'Test Item' });
    expect(response.status).toBe(201);
  });
});
```

## Guidelines

1. **Isolation**: Each test creates its own data; don't rely on shared state
2. **Cleanup**: Always clean up created resources in `afterAll`
3. **Deterministic**: Fixtures should produce consistent, predictable data
4. **Environment-agnostic**: Work against any environment (dev, staging, prod)
