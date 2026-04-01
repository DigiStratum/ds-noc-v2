/**
 * E2E Tests for Multi-Tenant Requirements
 * 
 * Covers: FR-TENANT-* requirements
 * @see REQUIREMENTS.md for full requirement details
 */
import { test, expect } from '@playwright/test';

test.describe('FR-TENANT-001: User session identifies current tenant', () => {
  test('should display current tenant in session', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/dashboard');
    
    // Current tenant should be displayed
    const tenantDisplay = page.locator('[data-testid="current-tenant"]');
    await expect(tenantDisplay).toBeVisible();
    await expect(tenantDisplay).not.toBeEmpty();
  });

  test('should show "Personal" when no tenant selected', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-no-tenant',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/dashboard');
    
    const tenantDisplay = page.locator('[data-testid="current-tenant"]');
    await expect(tenantDisplay).toContainText(/Personal|None/i);
  });
});

test.describe('FR-TENANT-002: Users with multiple tenants can switch via nav dropdown', () => {
  test('should display tenant switcher for multi-tenant users', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-multi-tenant',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/dashboard');
    
    // Tenant selector should be a dropdown
    const tenantSelector = page.locator('[data-testid="tenant-selector"]');
    await expect(tenantSelector).toBeVisible();
    
    // Click to open dropdown
    await tenantSelector.click();
    
    // Should show multiple tenant options
    const tenantOptions = page.locator('[data-testid="tenant-option"]');
    await expect(tenantOptions).toHaveCount({ minimum: 2 });
  });

  test('should switch tenant when option selected', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-multi-tenant',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/dashboard');
    
    const tenantSelector = page.locator('[data-testid="tenant-selector"]');
    await tenantSelector.click();
    
    // Select different tenant
    const secondTenant = page.locator('[data-testid="tenant-option"]').nth(1);
    const newTenantName = await secondTenant.textContent();
    await secondTenant.click();
    
    // Current tenant should update
    await expect(page.locator('[data-testid="current-tenant"]')).toContainText(newTenantName!);
  });
});

test.describe('FR-TENANT-003: All data queries are scoped to current tenant', () => {
  test('should only show data for current tenant', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-tenant-a',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/dashboard');
    
    // Data list should contain tenant-scoped items
    const dataItems = page.locator('[data-testid="data-list"] [data-tenant-id]');
    
    // All items should have same tenant ID
    const tenantIds = await dataItems.evaluateAll(
      items => items.map(item => item.getAttribute('data-tenant-id'))
    );
    
    // All should be same tenant (or empty list)
    if (tenantIds.length > 0) {
      const uniqueTenants = new Set(tenantIds);
      expect(uniqueTenants.size).toBe(1);
    }
  });
});

test.describe('FR-TENANT-004: API requests include X-Tenant-ID header', () => {
  test('should send X-Tenant-ID header with API requests', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-tenant-a',
      domain: '.digistratum.com',
      path: '/',
    }]);

    // Intercept API requests to verify header
    const apiRequests: { headers: Record<string, string> }[] = [];
    await page.route('**/api/**', async (route) => {
      apiRequests.push({ headers: route.request().headers() });
      await route.continue();
    });

    await page.goto('/dashboard');
    
    // Wait for API call
    await page.waitForTimeout(1000);
    
    // Verify X-Tenant-ID header was sent
    const hastenantHeader = apiRequests.some(
      req => 'x-tenant-id' in req.headers
    );
    expect(hastenantHeader).toBe(true);
  });
});
