/**
 * Test User Fixture
 *
 * Helpers for creating and managing test users in E2E tests.
 */

export interface TestUser {
  id: string;
  email: string;
  tenantId: string;
  accessToken: string;
}

export interface AuthenticatedClient {
  get: (path: string) => Promise<Response>;
  post: (path: string, body?: unknown) => Promise<Response>;
  put: (path: string, body?: unknown) => Promise<Response>;
  patch: (path: string, body?: unknown) => Promise<Response>;
  delete: (path: string) => Promise<Response>;
}

const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:3001';

/**
 * Create a test user with a unique email.
 * Uses TEST_USER_EMAIL and TEST_USER_PASSWORD from environment,
 * or generates a random user if not set.
 */
export async function createTestUser(): Promise<TestUser> {
  const testEmail = process.env.TEST_USER_EMAIL;
  const testPassword = process.env.TEST_USER_PASSWORD;

  if (testEmail && testPassword) {
    // Use pre-configured test user
    const response = await fetch(`${API_BASE_URL}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: testEmail, password: testPassword }),
    });

    if (!response.ok) {
      throw new Error(`Failed to authenticate test user: ${response.status}`);
    }

    const data = await response.json();
    return {
      id: data.user.id,
      email: testEmail,
      tenantId: data.user.tenantId,
      accessToken: data.accessToken,
    };
  }

  // Generate a unique test user
  const uniqueId = `test-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const email = `${uniqueId}@test.example.com`;
  const password = 'TestPassword123!';

  // Register new user
  const registerResponse = await fetch(`${API_BASE_URL}/api/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!registerResponse.ok) {
    throw new Error(`Failed to register test user: ${registerResponse.status}`);
  }

  const registerData = await registerResponse.json();
  return {
    id: registerData.user.id,
    email,
    tenantId: registerData.user.tenantId,
    accessToken: registerData.accessToken,
  };
}

/**
 * Create an authenticated HTTP client for a test user.
 */
export async function authenticateAs(user: TestUser): Promise<AuthenticatedClient> {
  const makeRequest = async (
    method: string,
    path: string,
    body?: unknown
  ): Promise<Response> => {
    const url = path.startsWith('http') ? path : `${API_BASE_URL}${path}`;
    const headers: Record<string, string> = {
      Authorization: `Bearer ${user.accessToken}`,
    };

    if (body) {
      headers['Content-Type'] = 'application/json';
    }

    return fetch(url, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  };

  return {
    get: (path: string) => makeRequest('GET', path),
    post: (path: string, body?: unknown) => makeRequest('POST', path, body),
    put: (path: string, body?: unknown) => makeRequest('PUT', path, body),
    patch: (path: string, body?: unknown) => makeRequest('PATCH', path, body),
    delete: (path: string) => makeRequest('DELETE', path),
  };
}

/**
 * Clean up a test user and their data.
 * Call this in afterAll() to ensure test isolation.
 */
export async function cleanupTestUser(user: TestUser): Promise<void> {
  // Only cleanup dynamically created users, not pre-configured ones
  if (process.env.TEST_USER_EMAIL === user.email) {
    return;
  }

  try {
    // Delete user account (this should cascade-delete user data)
    await fetch(`${API_BASE_URL}/api/auth/delete-account`, {
      method: 'DELETE',
      headers: {
        Authorization: `Bearer ${user.accessToken}`,
      },
    });
  } catch (error) {
    console.warn(`Failed to cleanup test user ${user.email}:`, error);
  }
}

/**
 * Create an unauthenticated HTTP client for testing 401 scenarios.
 */
export function unauthenticatedClient(): AuthenticatedClient {
  const makeRequest = async (
    method: string,
    path: string,
    body?: unknown
  ): Promise<Response> => {
    const url = path.startsWith('http') ? path : `${API_BASE_URL}${path}`;
    const headers: Record<string, string> = {};

    if (body) {
      headers['Content-Type'] = 'application/json';
    }

    return fetch(url, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  };

  return {
    get: (path: string) => makeRequest('GET', path),
    post: (path: string, body?: unknown) => makeRequest('POST', path, body),
    put: (path: string, body?: unknown) => makeRequest('PUT', path, body),
    patch: (path: string, body?: unknown) => makeRequest('PATCH', path, body),
    delete: (path: string) => makeRequest('DELETE', path),
  };
}
