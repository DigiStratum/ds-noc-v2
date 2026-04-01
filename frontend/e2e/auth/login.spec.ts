/**
 * E2E Tests for Login Flows
 *
 * @usecase UC-AUTH-LOGIN - User authenticates via DSAccount SSO
 * @requirement FR-AUTH-001 - Users authenticate via DSAccount SSO
 * @requirement FR-AUTH-002 - Unauthenticated requests redirect to SSO login
 */
import { test, expect } from '@playwright/test';

test.describe('@usecase UC-AUTH-LOGIN: User Login', () => {
  test.describe('@requirement FR-AUTH-001: Users authenticate via DSAccount SSO', () => {
    test('should redirect unauthenticated user to SSO', async ({ page }) => {
      // Navigate to a protected page
      await page.goto('/dashboard');

      // Should redirect to DSAccount SSO
      await expect(page).toHaveURL(/account\.digistratum\.com/);
    });

    test('should display user info after SSO callback', async ({ page, context }) => {
      // Set up authenticated session (mock SSO callback)
      await context.addCookies([
        {
          name: 'ds_session',
          value: 'test-session-token',
          domain: '.digistratum.com',
          path: '/',
        },
      ]);

      await page.goto('/dashboard');

      // User should be authenticated and see dashboard
      await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
    });
  });

  test.describe('@requirement FR-AUTH-002: Unauthenticated requests redirect to SSO login', () => {
    test('should redirect protected API requests', async ({ page }) => {
      // Clear any session
      await page.context().clearCookies();

      await page.goto('/dashboard');

      // Should redirect to SSO login
      await expect(page).toHaveURL(/account\.digistratum\.com.*login/);
    });
  });

  test.describe('@requirement FR-AUTH-003: Session includes user identity and tenant context', () => {
    test('should include tenant context in session', async ({ page, context }) => {
      // Set up authenticated session with tenant
      await context.addCookies([
        {
          name: 'ds_session',
          value: 'test-session-token',
          domain: '.digistratum.com',
          path: '/',
        },
      ]);

      await page.goto('/dashboard');

      // Tenant selector should show current tenant
      await expect(page.locator('[data-testid="tenant-selector"]')).toBeVisible();
    });
  });
});
