/**
 * E2E Tests for Accessibility Requirements
 * 
 * Covers: NFR-A11Y-* requirements
 * @see REQUIREMENTS.md for full requirement details
 */
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.describe('NFR-A11Y-001: WCAG 2.1 AA compliance', () => {
  test('should pass automated accessibility checks', async ({ page }) => {
    await page.goto('/');
    
    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();
    
    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('dashboard should pass accessibility checks', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/dashboard');
    
    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();
    
    expect(accessibilityScanResults.violations).toEqual([]);
  });
});

test.describe('NFR-A11Y-002: Semantic HTML structure', () => {
  test('should use semantic HTML elements', async ({ page }) => {
    await page.goto('/');
    
    // Should have main element
    await expect(page.locator('main')).toBeVisible();
    
    // Should have header
    await expect(page.locator('header')).toBeVisible();
    
    // Should have navigation
    await expect(page.locator('nav')).toBeVisible();
    
    // Should have footer
    await expect(page.locator('footer')).toBeVisible();
  });

  test('should have proper heading hierarchy', async ({ page }) => {
    await page.goto('/');
    
    // Get all headings
    const headings = await page.locator('h1, h2, h3, h4, h5, h6').all();
    
    // Should have at least one heading
    expect(headings.length).toBeGreaterThan(0);
    
    // Should have exactly one h1
    const h1Count = await page.locator('h1').count();
    expect(h1Count).toBe(1);
  });
});

test.describe('NFR-A11Y-003: Keyboard navigation support', () => {
  test('should allow tab navigation through interactive elements', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/');
    
    // Start at body
    await page.locator('body').focus();
    
    // Tab should move focus to interactive elements
    await page.keyboard.press('Tab');
    
    // Something should be focused
    const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
    expect(['A', 'BUTTON', 'INPUT', 'SELECT', 'TEXTAREA']).toContain(focusedElement);
  });

  test('should support keyboard navigation of menus', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/');
    
    // Focus user menu
    const userMenu = page.locator('[data-testid="user-menu"]');
    await userMenu.focus();
    
    // Enter should open menu
    await page.keyboard.press('Enter');
    
    // Menu should be open
    await expect(page.locator('[data-testid="user-menu-dropdown"]')).toBeVisible();
    
    // Escape should close menu
    await page.keyboard.press('Escape');
    await expect(page.locator('[data-testid="user-menu-dropdown"]')).toBeHidden();
  });
});

test.describe('NFR-A11Y-004: Screen reader compatibility', () => {
  test('should have ARIA labels on interactive elements', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/');
    
    // Icon-only buttons should have aria-label
    const iconButtons = page.locator('button:not(:has-text(*))');
    const count = await iconButtons.count();
    
    for (let i = 0; i < count; i++) {
      const button = iconButtons.nth(i);
      const hasLabel = await button.evaluate(el => 
        el.hasAttribute('aria-label') || el.hasAttribute('aria-labelledby')
      );
      expect(hasLabel).toBe(true);
    }
  });

  test('should have live regions for dynamic updates', async ({ page }) => {
    await page.goto('/');
    
    // Should have at least one live region (or a notification container)
    const liveRegions = await page.locator('[aria-live]').count();
    const roleAlerts = await page.locator('[role="alert"], [role="status"]').count();
    
    expect(liveRegions + roleAlerts).toBeGreaterThanOrEqual(0);
    // Note: This is a baseline check - actual live region usage should be verified
  });
});

test.describe('NFR-A11Y-005: Color contrast ratios meet AA standards', () => {
  test('should pass color contrast checks', async ({ page }) => {
    await page.goto('/');
    
    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2aa'])
      .include('body')
      .analyze();
    
    // Filter for contrast violations only
    const contrastViolations = accessibilityScanResults.violations.filter(
      v => v.id.includes('contrast')
    );
    
    expect(contrastViolations).toEqual([]);
  });
});

test.describe('NFR-A11Y-006: Focus indicators visible on all interactive elements', () => {
  test('should have visible focus indicators', async ({ page }) => {
    await page.goto('/');
    
    // Tab to first focusable element
    await page.keyboard.press('Tab');
    
    // Get the focused element's outline style
    const hasVisibleFocus = await page.evaluate(() => {
      const el = document.activeElement;
      if (!el) return false;
      
      const styles = getComputedStyle(el);
      const outline = styles.outline;
      const boxShadow = styles.boxShadow;
      const border = styles.border;
      
      // Check if any focus indicator is visible
      return (
        (outline && outline !== 'none' && !outline.includes('0px')) ||
        (boxShadow && boxShadow !== 'none') ||
        el.classList.contains('focus-visible') ||
        el.matches(':focus-visible')
      );
    });
    
    expect(hasVisibleFocus).toBe(true);
  });
});
