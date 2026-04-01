/**
 * App Configuration
 * 
 * Customize this file for your app.
 * Template updates preserve this file.
 */
import type { MenuItem } from '@digistratum/layout';

export interface AppConfig {
  /** App identifier (used for app-switcher highlighting) */
  id: string;
  /** Display name */
  name: string;
  /** App logo URL (optional - uses ecosystem default if not set) */
  logo?: string;
  /** Base URL for this app (computed at runtime from current origin) */
  baseUrl: string;
  /** SSO provider URL (computed at runtime from ecosystem) */
  ssoUrl: string;
  /** App-specific menu items */
  menuItems?: MenuItem[];
  /** Footer links */
  footerLinks?: { label: string; url: string }[];
  /** Feature flags */
  features?: Record<string, boolean>;
}

/**
 * Select ecosystem logo based on hostname.
 * - *.leapkick.com -> lk-logo.svg
 * - *.digistratum.com (and everything else) -> ds-logo.svg
 */
function getEcosystemLogo(): string {
  const hostname = typeof window !== 'undefined' ? window.location.hostname : '';
  if (hostname.endsWith('.leapkick.com') || hostname === 'leapkick.com') {
    return '/lk-logo.svg';
  }
  return '/ds-logo.svg';
}

/**
 * Determine SSO URL based on current ecosystem.
 * - *.leapkick.com -> account.leapkick.com
 * - *.digistratum.com -> account.digistratum.com
 */
function getSsoUrl(): string {
  const hostname = typeof window !== 'undefined' ? window.location.hostname : '';
  if (hostname.endsWith('.leapkick.com') || hostname === 'leapkick.com') {
    return 'https://account.leapkick.com';
  }
  return 'https://account.digistratum.com';
}

/**
 * Get base URL from current window origin.
 * Falls back to placeholder for SSR/build-time contexts.
 */
function getBaseUrl(): string {
  if (typeof window !== 'undefined') {
    return window.location.origin;
  }
  // Fallback for SSR - will be replaced by actual origin on client
  return 'https://noc-v2.digistratum.com';
}

// TODO: Update these values for your app
const config: AppConfig = {
  id: 'dsnocv2',
  name: 'DS Noc V2',
  logo: getEcosystemLogo(),
  baseUrl: getBaseUrl(),
  ssoUrl: getSsoUrl(),
  menuItems: [
    // Add your app's navigation items here
    // { label: 'Dashboard', href: '/dashboard', icon: 'dashboard' },
  ],
  footerLinks: [
    // Add app-specific footer links here (Privacy/Terms/Support provided by DSAppShell)
  ],
  features: {
    // App-specific feature flags
  },
};

export default config;
