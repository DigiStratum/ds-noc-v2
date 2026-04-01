/**
 * Dashboard UI Tests (Authenticated)
 *
 * Example UI test using authenticatedPage fixture.
 * Demonstrates DS SSO cookie-based auth pattern.
 */

import { test, expect } from '../fixtures';

test.describe('Dashboard (Authenticated)', () => {
  /**
   * @covers FR-UI-001 Dashboard accessible to authenticated users
   */
  test('displays dashboard content', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/dashboard');

    // Should not redirect to login
    await expect(authenticatedPage).not.toHaveURL(/login/);

    // Should show dashboard content
    await expect(authenticatedPage.locator('main')).toBeVisible();
  });

  /**
   * @covers FR-UI-002 User info shown in header
   */
  test('shows user menu in header', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/dashboard');

    // User menu should be visible
    const userMenu = authenticatedPage.locator('[data-testid="user-menu"]');
    await expect(userMenu).toBeVisible();
  });

  /**
   * @covers FR-UI-003 Navigation available when authenticated
   */
  test('navigation menu is accessible', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/dashboard');

    // Navigation should be present
    const nav = authenticatedPage.locator('nav, [role="navigation"]');
    await expect(nav).toBeVisible();
  });
});
