/**
 * Example UI Test: Navigation Flow
 *
 * Demonstrates Playwright E2E test patterns with @covers markers.
 */

import { test, expect } from '@playwright/test';

const APP_BASE_URL = process.env.APP_BASE_URL || 'http://localhost:5173';

test.describe('Navigation', () => {
  /**
   * @covers FR-UI-001 Landing page accessible without auth
   */
  test('displays landing page', async ({ page }) => {
    await page.goto(APP_BASE_URL);

    await expect(page).toHaveTitle(/{{APP_DISPLAY_NAME}}/);
    await expect(page.locator('main')).toBeVisible();
  });

  /**
   * @covers FR-UI-002 Navigation menu visible to authenticated users
   */
  test('shows navigation menu when authenticated', async ({ page }) => {
    // Login first
    await page.goto(`${APP_BASE_URL}/login`);
    await page.fill('[data-testid="email-input"]', process.env.TEST_USER_EMAIL || 'test@example.com');
    await page.fill('[data-testid="password-input"]', process.env.TEST_USER_PASSWORD || 'password');
    await page.click('[data-testid="login-button"]');

    // Wait for redirect to dashboard
    await expect(page).toHaveURL(/dashboard/);

    // Check navigation elements
    await expect(page.locator('[data-testid="nav-menu"]')).toBeVisible();
    await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
  });

  /**
   * @covers FR-UI-003 Unauthenticated users redirected to login
   */
  test('redirects to login for protected routes', async ({ page }) => {
    await page.goto(`${APP_BASE_URL}/dashboard`);

    // Should redirect to login
    await expect(page).toHaveURL(/login/);
  });
});

test.describe('Accessibility', () => {
  /**
   * @covers NFR-A11Y-001 Keyboard navigation support
   */
  test('supports keyboard navigation', async ({ page }) => {
    await page.goto(APP_BASE_URL);

    // Tab through interactive elements
    await page.keyboard.press('Tab');
    const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
    expect(focusedElement).toBeTruthy();

    // Skip link should be first focusable element
    const skipLink = page.locator('[data-testid="skip-to-content"]');
    if (await skipLink.isVisible()) {
      await expect(skipLink).toBeFocused();
    }
  });

  /**
   * @covers NFR-A11Y-002 Screen reader landmarks present
   */
  test('has proper ARIA landmarks', async ({ page }) => {
    await page.goto(APP_BASE_URL);

    // Check for main landmark
    await expect(page.locator('main, [role="main"]')).toBeVisible();

    // Check for navigation landmark
    await expect(page.locator('nav, [role="navigation"]')).toBeVisible();
  });
});

test.describe('Error States', () => {
  /**
   * @covers FR-UI-004 404 page for invalid routes
   */
  test('displays 404 page for unknown routes', async ({ page }) => {
    await page.goto(`${APP_BASE_URL}/non-existent-page-12345`);

    await expect(page.locator('text=/not found|404/i')).toBeVisible();
  });

  /**
   * @covers FR-UI-005 Error boundary catches crashes
   */
  test('shows error boundary on component crash', async ({ page }) => {
    // Navigate to a page that triggers an error (test-only route)
    await page.goto(`${APP_BASE_URL}/__test/crash`);

    // Should show error UI, not crash the page
    await expect(page.locator('text=/something went wrong|error/i')).toBeVisible();
  });
});
