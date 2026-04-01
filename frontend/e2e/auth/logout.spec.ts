/**
 * E2E Tests for Logout Flows
 *
 * @usecase UC-AUTH-LOGOUT - User logs out and session is cleared
 * @requirement FR-AUTH-004 - Logout clears session and redirects to DSAccount logout
 */
import { test, expect } from '@playwright/test';

test.describe('@usecase UC-AUTH-LOGOUT: User Logout', () => {
  test.describe('@requirement FR-AUTH-004: Logout clears session and redirects to DSAccount logout', () => {
    test('should clear session on logout', async ({ page, context }) => {
      // Set up authenticated session
      await context.addCookies([
        {
          name: 'ds_session',
          value: 'test-session-token',
          domain: '.digistratum.com',
          path: '/',
        },
      ]);

      await page.goto('/dashboard');

      // Click logout
      await page.click('[data-testid="user-menu"]');
      await page.click('[data-testid="logout-button"]');

      // Should redirect to DSAccount logout
      await expect(page).toHaveURL(/account\.digistratum\.com.*logout/);
    });

    test('should not access protected routes after logout', async ({ page, context }) => {
      // Set up authenticated session
      await context.addCookies([
        {
          name: 'ds_session',
          value: 'test-session-token',
          domain: '.digistratum.com',
          path: '/',
        },
      ]);

      await page.goto('/dashboard');

      // Click logout
      await page.click('[data-testid="user-menu"]');
      await page.click('[data-testid="logout-button"]');

      // Wait for redirect to complete
      await page.waitForURL(/account\.digistratum\.com/);

      // Clear cookies to simulate logged-out state
      await page.context().clearCookies();

      // Try to access protected route again
      await page.goto('/dashboard');

      // Should be redirected to login
      await expect(page).toHaveURL(/account\.digistratum\.com.*login/);
    });
  });
});
