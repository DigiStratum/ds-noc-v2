/**
 * E2E Tests for Performance Requirements
 * 
 * Covers: NFR-PERF-* requirements
 * @see REQUIREMENTS.md for full requirement details
 */
import { test, expect } from '@playwright/test';

test.describe('NFR-PERF-001: Page load time < 3 seconds', () => {
  test('should load home page within 3 seconds', async ({ page }) => {
    const startTime = Date.now();
    
    await page.goto('/', { waitUntil: 'load' });
    
    const loadTime = Date.now() - startTime;
    expect(loadTime).toBeLessThan(3000);
  });

  test('should load dashboard within 3 seconds', async ({ page, context }) => {
    await context.addCookies([{
      name: 'ds_session',
      value: 'test-session-token',
      domain: '.digistratum.com',
      path: '/',
    }]);

    const startTime = Date.now();
    
    await page.goto('/dashboard', { waitUntil: 'load' });
    
    const loadTime = Date.now() - startTime;
    expect(loadTime).toBeLessThan(3000);
  });
});

test.describe('NFR-PERF-002: API response time < 500ms (p95)', () => {
  test('should respond to health check quickly', async ({ page }) => {
    const startTime = Date.now();
    
    const response = await page.request.get('/api/health');
    
    const responseTime = Date.now() - startTime;
    expect(responseTime).toBeLessThan(500);
    expect(response.ok()).toBe(true);
  });
});

test.describe('NFR-PERF-003: Time to interactive < 2 seconds', () => {
  test('should become interactive within 2 seconds', async ({ page }) => {
    const tti = await page.evaluate(async () => {
      return new Promise((resolve) => {
        const start = performance.now();
        
        // Check if page is interactive by looking for hydration
        const checkInteractive = () => {
          const buttons = document.querySelectorAll('button');
          const isInteractive = buttons.length > 0;
          
          if (isInteractive) {
            resolve(performance.now() - start);
          } else {
            requestAnimationFrame(checkInteractive);
          }
        };
        
        checkInteractive();
      });
    });
    
    await page.goto('/');
    
    // Note: This is a simplified check - real TTI should use Lighthouse
    expect(tti).toBeLessThan(2000);
  });
});

test.describe('NFR-PERF-004: Core Web Vitals meet good thresholds', () => {
  test('should have acceptable Largest Contentful Paint', async ({ page }) => {
    await page.goto('/');
    
    // Wait for LCP
    const lcp = await page.evaluate(async () => {
      return new Promise((resolve) => {
        new PerformanceObserver((entryList) => {
          const entries = entryList.getEntries();
          const lastEntry = entries[entries.length - 1];
          resolve(lastEntry.startTime);
        }).observe({ type: 'largest-contentful-paint', buffered: true });
        
        // Fallback timeout
        setTimeout(() => resolve(2500), 3000);
      });
    });
    
    // LCP should be < 2.5s for "good" rating
    expect(lcp).toBeLessThan(2500);
  });

  test('should have minimal Cumulative Layout Shift', async ({ page }) => {
    await page.goto('/');
    
    // Wait for page to stabilize
    await page.waitForTimeout(1000);
    
    const cls = await page.evaluate(() => {
      return new Promise((resolve) => {
        let clsValue = 0;
        
        new PerformanceObserver((entryList) => {
          for (const entry of entryList.getEntries()) {
            if (!(entry as any).hadRecentInput) {
              clsValue += (entry as any).value;
            }
          }
        }).observe({ type: 'layout-shift', buffered: true });
        
        // Give time to collect CLS
        setTimeout(() => resolve(clsValue), 500);
      });
    });
    
    // CLS should be < 0.1 for "good" rating
    expect(cls).toBeLessThan(0.1);
  });
});
