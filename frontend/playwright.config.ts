import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E test configuration for DS App Template
 *
 * Configuration aligned with DS ecosystem requirements:
 * - baseURL from APP_URL env var (matches DS deploy patterns)
 * - Screenshot/video on failure for debugging
 * - Trace on first retry for CI debugging
 * - Chromium-first (start simple, expand as needed)
 *
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html'],
    ['json', { outputFile: 'e2e-results.json' }],
  ],

  use: {
    // APP_URL is the standard DS env var for app public URL
    baseURL: process.env.APP_URL || 'http://localhost:5173',

    // Debugging artifacts
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',

    // Headless by default (CI-compatible)
    headless: true,
  },

  projects: [
    // Smoke tests - fast, blocks deploy, runs first
    {
      name: 'smoke',
      testMatch: '**/smoke.spec.ts',
      use: { ...devices['Desktop Chrome'] },
      timeout: 30000, // 30s max per test
    },
    // Primary: Chromium (start simple per issue requirements)
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    // Optional: Firefox
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    // Optional: WebKit
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
    // Optional: Mobile
    {
      name: 'mobile-chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'mobile-safari',
      use: { ...devices['iPhone 12'] },
    },
  ],

  webServer: {
    command: 'pnpm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
});
