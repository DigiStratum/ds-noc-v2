/**
 * E2E Tests for Theming Requirements
 * 
 * Covers: FR-THEME-* requirements
 * @see REQUIREMENTS.md for full requirement details
 */
import { test, expect } from '@playwright/test';

test.describe('FR-THEME-001: Light and dark theme options', () => {
  test('should support light theme', async ({ page }) => {
    await page.goto('/');
    
    // Set light theme preference
    await page.evaluate(() => {
      document.documentElement.classList.remove('dark');
      document.documentElement.classList.add('light');
    });
    
    // Body should have light theme styling
    const backgroundColor = await page.evaluate(() => {
      return getComputedStyle(document.body).backgroundColor;
    });
    
    // Light theme should have light background (RGB values > 200)
    const rgb = backgroundColor.match(/\d+/g)?.map(Number);
    expect(rgb?.[0]).toBeGreaterThan(200);
  });

  test('should support dark theme', async ({ page }) => {
    await page.goto('/');
    
    // Set dark theme preference
    await page.evaluate(() => {
      document.documentElement.classList.remove('light');
      document.documentElement.classList.add('dark');
    });
    
    // Body should have dark theme styling
    const backgroundColor = await page.evaluate(() => {
      return getComputedStyle(document.body).backgroundColor;
    });
    
    // Dark theme should have dark background (RGB values < 60)
    const rgb = backgroundColor.match(/\d+/g)?.map(Number);
    expect(rgb?.[0]).toBeLessThan(60);
  });
});

test.describe('FR-THEME-002: Theme preference stored in user session', () => {
  test('should persist theme preference across page loads', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/');
    
    // Toggle to dark theme
    await page.click('[data-testid="theme-toggle"]');
    
    // Verify dark mode activated
    await expect(page.locator('html')).toHaveClass(/dark/);
    
    // Reload page
    await page.reload();
    
    // Theme should persist
    await expect(page.locator('html')).toHaveClass(/dark/);
  });
});

test.describe('FR-THEME-003: Theme applied via CSS variables', () => {
  test('should use CSS variables for theming', async ({ page }) => {
    await page.goto('/');
    
    // Check that CSS variables are used
    const usesVariables = await page.evaluate(() => {
      const styles = getComputedStyle(document.documentElement);
      // Check for common theme variables
      const bgVar = styles.getPropertyValue('--background');
      const fgVar = styles.getPropertyValue('--foreground');
      return bgVar !== '' || fgVar !== '';
    });
    
    expect(usesVariables).toBe(true);
  });

  test('should change CSS variables when theme changes', async ({ page }) => {
    await page.goto('/');
    
    // Get initial variable values
    const lightBg = await page.evaluate(() => {
      document.documentElement.classList.remove('dark');
      document.documentElement.classList.add('light');
      return getComputedStyle(document.documentElement).getPropertyValue('--background');
    });
    
    // Switch to dark theme
    const darkBg = await page.evaluate(() => {
      document.documentElement.classList.remove('light');
      document.documentElement.classList.add('dark');
      return getComputedStyle(document.documentElement).getPropertyValue('--background');
    });
    
    // Variables should be different
    expect(lightBg).not.toEqual(darkBg);
  });
});
