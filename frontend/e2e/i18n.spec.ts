/**
 * E2E Tests for Internationalization Requirements
 * 
 * Covers: FR-I18N-* requirements
 * @see REQUIREMENTS.md for full requirement details
 */
import { test, expect } from '@playwright/test';

test.describe('FR-I18N-001: Static strings loaded from language packs', () => {
  test('should load default language strings', async ({ page }) => {
    await page.goto('/');
    
    // UI text should be present (not raw keys)
    const mainContent = await page.textContent('body');
    
    // Should not contain raw i18n keys like "common.welcome" or "nav.home"
    expect(mainContent).not.toMatch(/\b[a-z]+\.[a-z]+\.[a-z]+\b/);
  });

  test('should switch language when changed', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/');
    
    // Get initial text
    const initialText = await page.textContent('[data-testid="nav-home"]');
    
    // Change language
    await page.click('[data-testid="language-selector"]');
    await page.click('[data-testid="language-option-es"]');
    
    // Text should change
    const newText = await page.textContent('[data-testid="nav-home"]');
    expect(newText).not.toEqual(initialText);
  });
});

test.describe('FR-I18N-002: Dynamic content translated and cached on-the-fly', () => {
  test.skip('should translate dynamic content', async ({ page }) => {
    // This test is skipped until dynamic translation is implemented
    // Requirement is marked ❌ Not implemented in REQUIREMENTS.md
  });
});

test.describe('FR-I18N-003: Language preference stored in user session', () => {
  test('should persist language preference across page loads', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    await page.goto('/');
    
    // Change to Spanish
    await page.click('[data-testid="language-selector"]');
    await page.click('[data-testid="language-option-es"]');
    
    // Reload page
    await page.reload();
    
    // Language should still be Spanish
    const lang = await page.getAttribute('html', 'lang');
    expect(lang).toBe('es');
  });
});
