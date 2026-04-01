# FR-NAV: Navigation

> Standard DS ecosystem navigation components.
> Consistent header, footer, and app-switching across all DS apps.

---

## Requirements

### FR-NAV-001: Standard header layout

All DS apps use a consistent header with logo, navigation, tenant switcher, and user menu.

**Acceptance Criteria:**
1. Logo in upper-left corner, links to app home
2. Primary navigation links horizontally aligned in header
3. Tenant switcher visible for multi-tenant users (see FR-TENANT-002)
4. User menu in upper-right with profile, settings, logout options
5. Header height consistent across all DS apps (64px desktop, 56px mobile)
6. Header remains fixed/sticky on scroll

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `frontend/src/components/Header.test.tsx` |
| E2E test | `frontend/e2e/navigation.spec.ts:header elements` |
| Visual test | Chromatic snapshot (if configured) |

**Evidence:**
- CI test results
- Screenshot comparison across DS apps

---

### FR-NAV-002: App-switcher shows available DS ecosystem apps

Users can navigate between DS ecosystem apps via an app-switcher component.

**Acceptance Criteria:**
1. App-switcher accessible from header (grid icon or similar)
2. Shows all DS ecosystem apps user has access to
3. Current app visually highlighted
4. Links open in same tab (not new window)
5. Apps shown: DSAccount, DS Projects, DS CRM, DS Developer (based on permissions)
6. App icons/logos displayed consistently

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `frontend/src/components/AppSwitcher.test.tsx` |
| E2E test | `frontend/e2e/navigation.spec.ts:app switcher` |

**Evidence:**
- CI test results
- Manual verification: click app-switcher → navigate to another DS app

---

### FR-NAV-003: Footer with copyright and standard links

Consistent footer across all DS apps with legal/support links.

**Acceptance Criteria:**
1. Copyright notice: "© {year} DigiStratum. All rights reserved."
2. Links: Privacy Policy, Terms of Service, Support/Help
3. Footer sticks to bottom of viewport when content is short
4. Footer below content when content exceeds viewport
5. Responsive: links stack vertically on mobile

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `frontend/src/components/Footer.test.tsx` |
| E2E test | `frontend/e2e/navigation.spec.ts:footer elements` |

**Evidence:**
- CI test results
- Visual inspection on short and long pages

---

### FR-NAV-004: Mobile-responsive layout

Navigation adapts to mobile viewports with hamburger menu pattern.

**Acceptance Criteria:**
1. Breakpoint at 768px (tablet/mobile boundary)
2. Desktop: horizontal nav links visible in header
3. Mobile: hamburger icon replaces nav links
4. Hamburger opens slide-out menu with all navigation
5. Menu closes on link click or outside tap
6. Tenant switcher and user menu accessible in mobile menu
7. Touch targets minimum 44x44px (accessibility)

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `frontend/src/components/MobileNav.test.tsx` |
| E2E test | `frontend/e2e/navigation.spec.ts:mobile navigation` |
| E2E test | `frontend/e2e/accessibility.spec.ts:touch targets` |

**Evidence:**
- CI test results (Playwright mobile viewport)
- Manual testing on actual mobile device

---

## Implementation

### Header Component

```tsx
// frontend/src/components/Header.tsx
export function Header() {
  const { user } = useAuth();
  const { currentTenant, tenants } = useTenant();
  const isMobile = useMediaQuery('(max-width: 768px)');
  
  return (
    <header className="h-16 md:h-14 fixed top-0 w-full bg-background border-b">
      <div className="container flex items-center justify-between h-full">
        <Logo />
        
        {isMobile ? (
          <MobileNavToggle />
        ) : (
          <nav aria-label="Main navigation">
            <NavLinks />
          </nav>
        )}
        
        <div className="flex items-center gap-4">
          <AppSwitcher />
          {tenants.length > 1 && <TenantSwitcher />}
          <UserMenu user={user} />
        </div>
      </div>
    </header>
  );
}
```

### Skip Link (Accessibility)

```tsx
// Include at top of page layout
<a 
  href="#main" 
  className="skip-link sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2"
>
  Skip to main content
</a>
```

### Footer Component

```tsx
// frontend/src/components/Footer.tsx
export function Footer() {
  const year = new Date().getFullYear();
  
  return (
    <footer className="border-t py-6 mt-auto">
      <div className="container flex flex-col md:flex-row justify-between items-center gap-4">
        <p className="text-sm text-muted-foreground">
          © {year} DigiStratum. All rights reserved.
        </p>
        <nav className="flex gap-4 text-sm">
          <a href="https://digistratum.com/privacy">Privacy Policy</a>
          <a href="https://digistratum.com/terms">Terms of Service</a>
          <a href="https://digistratum.com/support">Support</a>
        </nav>
      </div>
    </footer>
  );
}
```

---

## Shared Package

Navigation components are provided by `@digistratum/layout`:

```tsx
import { AppShell, Header, Footer, AppSwitcher } from '@digistratum/layout';

function App() {
  return (
    <AppShell header={<Header />} footer={<Footer />}>
      <main id="main">{/* Page content */}</main>
    </AppShell>
  );
}
```

---

## Traceability

| Requirement | Implementation | Test | Status |
|-------------|----------------|------|--------|
| FR-NAV-001 | `@digistratum/layout:Header` | `Header.test.tsx`, `navigation.spec.ts` | ⚠️ |
| FR-NAV-002 | `@digistratum/layout:AppSwitcher` | `AppSwitcher.test.tsx`, `navigation.spec.ts` | ⚠️ |
| FR-NAV-003 | `@digistratum/layout:Footer` | `Footer.test.tsx`, `navigation.spec.ts` | ⚠️ |
| FR-NAV-004 | `@digistratum/layout:MobileNav` | `MobileNav.test.tsx`, `navigation.spec.ts` | ⚠️ |

---

*Last updated: 2026-03-23*
