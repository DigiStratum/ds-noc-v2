/**
 * E2E Tests for Navigation Requirements
 * 
 * Covers: FR-NAV-* requirements
 * @see REQUIREMENTS.md for full requirement details
 */
import { test, expect } from '@playwright/test';

test.describe('FR-NAV-001: Standard header with logo, nav links, tenant switcher, user menu', () => {
  test('should display header with all required elements', async ({ page, context }) => {
    // Set up authenticated session
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/');
    
    // Header should contain logo (upper-left)
    const logo = page.locator('header [data-testid="logo"]');
    await expect(logo).toBeVisible();
    
    // Nav links should be present
    const nav = page.locator('header nav');
    await expect(nav).toBeVisible();
    
    // Tenant switcher should be present
    const tenantSwitcher = page.locator('[data-testid="tenant-selector"]');
    await expect(tenantSwitcher).toBeVisible();
    
    // User menu should be present
    const userMenu = page.locator('[data-testid="user-menu"]');
    await expect(userMenu).toBeVisible();
  });

  test('logo should be in upper-left position', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/');
    
    const logo = page.locator('header [data-testid="logo"]');
    const box = await logo.boundingBox();
    
    // Logo should be near top-left
    expect(box?.x).toBeLessThan(100);
    expect(box?.y).toBeLessThan(100);
  });
});

test.describe('FR-NAV-002: App-switcher shows available DS ecosystem apps', () => {
  test('should display app switcher with ecosystem apps', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/');
    
    // Click app switcher
    await page.click('[data-testid="app-switcher"]');
    
    // Should show available apps
    const appMenu = page.locator('[data-testid="app-menu"]');
    await expect(appMenu).toBeVisible();
    
    // Should contain at least DSAccount link
    await expect(appMenu.locator('a[href*="account.digistratum.com"]')).toBeVisible();
  });
});

test.describe('FR-NAV-003: Footer with copyright and standard links', () => {
  test('should display footer with required elements', async ({ page }) => {
    await page.goto('/');
    
    const footer = page.locator('footer');
    await expect(footer).toBeVisible();
    
    // Should have copyright
    await expect(footer.locator('text=/©|Copyright/')).toBeVisible();
    
    // Should have standard links
    await expect(footer.locator('a')).toHaveCount({ minimum: 1 });
  });
});

test.describe('FR-NAV-004: Mobile-responsive layout', () => {
  test('should display hamburger menu on mobile', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    // Desktop nav should be hidden
    const desktopNav = page.locator('header nav.desktop-nav');
    await expect(desktopNav).toBeHidden();
    
    // Mobile hamburger should be visible
    const mobileMenuButton = page.locator('[data-testid="mobile-menu-button"]');
    await expect(mobileMenuButton).toBeVisible();
  });

  test('should expand mobile menu on hamburger click', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    // Click hamburger
    await page.click('[data-testid="mobile-menu-button"]');
    
    // Mobile menu should expand
    const mobileMenu = page.locator('[data-testid="mobile-menu"]');
    await expect(mobileMenu).toBeVisible();
  });
});
