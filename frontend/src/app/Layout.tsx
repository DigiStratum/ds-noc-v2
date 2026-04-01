import { ReactNode } from 'react';
import { AppShell, type FooterLink, type AuthContext, type ThemeContext } from '@digistratum/layout';
import { useAuth, useTheme } from '@digistratum/ds-core';
import config from './config';

interface LayoutProps {
  children: ReactNode;
  appName?: string;
  appLogo?: string;
  currentAppId?: string;
  extraFooterLinks?: FooterLink[];
  showAppSwitcher?: boolean;
  showUserMenu?: boolean;
  showGdprBanner?: boolean;
  /** Custom header content rendered above the standard header */
  customHeader?: ReactNode;
  /** Ad slot content between header and main content */
  headerAdSlot?: ReactNode;
  /** Ad slot content between main content and footer */
  footerAdSlot?: ReactNode;
}

/**
 * Empty placeholder for ad slots - renders 0-height div with data attribute
 * for potential ad injection while not affecting visible layout.
 */
function EmptyAdSlot({ position }: { position: 'header' | 'footer' }) {
  return (
    <div 
      data-ad-slot={position}
      aria-hidden="true"
      style={{ height: 0, overflow: 'hidden' }}
    />
  );
}

/**
 * Empty placeholder for custom header zone - renders 0-height div
 * for potential header injection while not affecting visible layout.
 */
function EmptyCustomHeader() {
  return (
    <div 
      data-custom-header
      aria-hidden="true"
      style={{ height: 0, overflow: 'hidden' }}
    />
  );
}

/**
 * Standard layout wrapper using AppShell
 * 
 * This provides:
 * - Custom header zone (above standard header, hidden by default)
 * - Header with navigation, user menu
 * - Header ad slot (between header and content, hidden by default)
 * - Main content area
 * - Footer ad slot (between content and footer, hidden by default)
 * - Footer with legal links
 * - GDPR banner (if enabled)
 * 
 * Ad slots and custom header render as 0-height placeholders by default.
 * Apps can inject visible content by passing customHeader, headerAdSlot, or footerAdSlot props.
 */
export function Layout({ 
  children, 
  appName = config.name, 
  appLogo = config.logo,
  currentAppId = config.id,
  extraFooterLinks = [],
  showAppSwitcher = true,
  showUserMenu = true,
  showGdprBanner = true,
  customHeader,
  headerAdSlot,
  footerAdSlot,
}: LayoutProps) {
  const { user, login, logout, isAuthenticated } = useAuth();
  const { theme, setTheme, resolvedTheme } = useTheme();

  // Combine default footer links with extra links
  const allFooterLinks: FooterLink[] = [
    ...(config.footerLinks || []),
    ...extraFooterLinks,
  ];

  // Build auth context for AppShell
  const authContext: AuthContext = {
    user: user ? {
      id: user.id,
      email: user.email,
      name: user.name,
    } : null,
    isAuthenticated,
    currentTenant: null,
    login: () => login(),
    logout,
    switchTenant: () => {}, // Not implemented in basic layout
  };

  // Build theme context for AppShell
  const themeContext: ThemeContext = {
    theme: theme as 'light' | 'dark' | 'system',
    resolvedTheme: resolvedTheme as 'light' | 'dark',
    setTheme,
  };

  return (
    <AppShell
      appName={appName}
      currentAppId={currentAppId}
      logoUrl={appLogo}
      auth={authContext}
      theme={themeContext}
      footerLinks={allFooterLinks}
      copyrightHolder="DigiStratum, LLC"
      showAppSwitcher={showAppSwitcher}
      showThemeToggle={false}
      showUserMenu={showUserMenu}
      showGdprBanner={showGdprBanner}
      customHeader={customHeader ?? <EmptyCustomHeader />}
      headerAdSlot={headerAdSlot ?? <EmptyAdSlot position="header" />}
      footerAdSlot={footerAdSlot ?? <EmptyAdSlot position="footer" />}
    >
      {children}
    </AppShell>
  );
}

export default Layout;
