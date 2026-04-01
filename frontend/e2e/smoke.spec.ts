/**
 * Smoke Tests - Critical Path Verification
 *
 * These tests verify the most critical user paths and should complete in <2 minutes.
 * Smoke tests block deployment - if these fail, the app should not be deployed.
 *
 * @smoke - Tag for CI to identify deploy-blocking tests
 *
 * Critical paths covered:
 * - App loads and renders
 * - Authentication flow (SSO redirect)
 * - Navigation renders correctly
 * - Core UI elements present
 */
import { test, expect } from '@playwright/test';

// Mark all tests in this file as smoke tests
test.describe('@smoke Critical Path Tests', () => {
  test.describe.configure({ timeout: 30000 }); // 30s max per test

  test('app loads without errors', async ({ page }) => {
    // Listen for console errors
    const errors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    await page.goto('/');

    // Page should load
    await expect(page).toHaveTitle(/.+/);

    // No critical JS errors
    const criticalErrors = errors.filter(
      (e) => !e.includes('favicon') && !e.includes('404')
    );
    expect(criticalErrors).toHaveLength(0);
  });

  test('unauthenticated user is redirected to SSO', async ({ page }) => {
    // @usecase UC-AUTH-LOGIN
    // @requirement FR-AUTH-001
    await page.goto('/dashboard');

    // Should redirect to DSAccount
    await expect(page).toHaveURL(/account\.digistratum\.com/);
  });

  test('authenticated user sees main layout', async ({ page, context }) => {
    // @usecase UC-NAV-VIEW
    // @requirement FR-NAV-001
    await context.addCookies([
      {
        name: 'ds_session',
        value: 'test-session-token',
        domain: '.digistratum.com',
        path: '/',
      },
    ]);

    await page.goto('/');

    // Header should be visible
    await expect(page.locator('header')).toBeVisible();

    // Footer should be visible
    await expect(page.locator('footer')).toBeVisible();
  });

  test('navigation elements render', async ({ page, context }) => {
    // @requirement FR-NAV-001
    await context.addCookies([
      {
        name: 'ds_session',
        value: 'test-session-token',
        domain: '.digistratum.com',
        path: '/',
      },
    ]);

    await page.goto('/');

    // Logo present
    await expect(page.locator('[data-testid="logo"]')).toBeVisible();

    // User menu present (indicates auth working)
    await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
  });

  test('app responds to route changes', async ({ page, context }) => {
    await context.addCookies([
      {
        name: 'ds_session',
        value: 'test-session-token',
        domain: '.digistratum.com',
        path: '/',
      },
    ]);

    await page.goto('/');

    // Navigate to another route
    await page.goto('/dashboard');

    // Page should respond (not hang or crash)
    await expect(page.locator('body')).toBeVisible();
  });

  test('critical API endpoint responds', async ({ page, context }) => {
    // @requirement FR-API-001
    await context.addCookies([
      {
        name: 'ds_session',
        value: 'test-session-token',
        domain: '.digistratum.com',
        path: '/',
      },
    ]);

    // Health endpoint should respond
    const response = await page.request.get('/api/health');
    expect(response.status()).toBeLessThan(500);
  });
});
