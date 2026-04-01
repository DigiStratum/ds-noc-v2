# Use Cases - {{APP_NAME}}

> User-centric scenario registry for tracking functional outcomes and E2E alignment.
> Complements REQUIREMENTS.md: use cases describe **what users do**, requirements specify **how it works**.
> E2E tests link to use case IDs, ensuring test coverage maps to user outcomes.

<!--
================================================================================
USE CASES TEMPLATE INSTRUCTIONS
================================================================================

1. GETTING STARTED
   - Replace {{APP_NAME}} with your application name
   - Keep standard UC-* sections as provided
   - Add app-specific use cases in designated areas
   - Remove this instruction block before publishing

2. USE CASE ID FORMAT
   UC-{CATEGORY}-{NNN}
   
   Standard categories:
   - UC-AUTH-*     Authentication flows
   - UC-NAV-*      Navigation and layout
   - UC-THEME-*    Theming and appearance
   - UC-TENANT-*   Multi-tenancy workflows
   - UC-APP-*      Application-specific (add yours here)

3. USE CASE STRUCTURE
   Each use case includes:
   - Actor: Who performs the action
   - Preconditions: What must be true before
   - Flow: Step-by-step user actions
   - Postconditions: Expected outcome
   - Requirements: Linked FR-*/NFR-* IDs
   - E2E Tests: File paths and test names

4. STATUS LEGEND
   - ✅ Complete: Implemented + E2E coverage
   - ⚠️ Partial: Implemented, missing E2E or edge cases
   - ❌ Not implemented
   - 🚧 In progress

5. NON-BREAKING CHANGE POLICY
   - New use cases: Always append (new IDs)
   - Existing use cases: Expand, never remove
   - Deprecated: Mark with ❌ DEPRECATED, do not delete
   - E2E tests lock use cases once covered

6. E2E TEST LINKING
   E2E tests reference use case IDs in describe blocks:
   
   test.describe('UC-AUTH-001: User logs in via SSO', () => {
     test('redirects to DSAccount login', async ({ page }) => {...});
   });
================================================================================
-->

---

## Status Legend

| Status | Meaning |
|--------|---------|
| ✅ | Complete: Implemented with E2E coverage |
| ⚠️ | Partial: Implemented, missing E2E or edge cases |
| ❌ | Not implemented |
| 🚧 | In progress |

---

## Use Case Summary

| ID | Use Case | Status | E2E Test |
|----|----------|--------|----------|
| **Authentication** |
| UC-AUTH-001 | User logs in via SSO | ⚠️ | `auth.spec.ts` |
| UC-AUTH-002 | User logs out | ⚠️ | `auth.spec.ts` |
| UC-AUTH-003 | Session expires and user re-authenticates | ⚠️ | `auth.spec.ts` |
| **Navigation** |
| UC-NAV-001 | User navigates using header menu | ⚠️ | `navigation.spec.ts` |
| UC-NAV-002 | User switches between DS ecosystem apps | ⚠️ | `navigation.spec.ts` |
| UC-NAV-003 | User accesses footer links | ⚠️ | `navigation.spec.ts` |
| UC-NAV-004 | User navigates on mobile device | ⚠️ | `navigation.spec.ts` |
| **Theming** |
| UC-THEME-001 | User switches between light and dark theme | ⚠️ | `theme.spec.ts` |
| UC-THEME-002 | User's theme preference persists across sessions | ⚠️ | `theme.spec.ts` |
| **Multi-Tenancy** |
| UC-TENANT-001 | User views data scoped to current tenant | ⚠️ | `tenant.spec.ts` |
| UC-TENANT-002 | User switches between tenants | ⚠️ | `tenant.spec.ts` |

---

## UC-AUTH: Authentication Use Cases

### UC-AUTH-001: User logs in via SSO

| Attribute | Value |
|-----------|-------|
| **Actor** | Unauthenticated User |
| **Status** | ⚠️ |
| **Requirements** | FR-AUTH-001, FR-AUTH-002 |
| **E2E Test** | `e2e/auth.spec.ts` |

**Preconditions:**
- User has a DSAccount
- User is not currently authenticated

**Flow:**
1. User navigates to any protected page
2. System redirects to DSAccount login
3. User enters credentials on DSAccount
4. DSAccount redirects back with session cookie
5. System displays the requested page

**Postconditions:**
- User is authenticated
- `ds_session` cookie is set
- User menu shows identity

**E2E Test Reference:**
```typescript
test.describe('UC-AUTH-001: User logs in via SSO', () => {
  test('should redirect unauthenticated user to SSO', ...);
  test('should display user info after SSO callback', ...);
});
```

---

### UC-AUTH-002: User logs out

| Attribute | Value |
|-----------|-------|
| **Actor** | Authenticated User |
| **Status** | ⚠️ |
| **Requirements** | FR-AUTH-004 |
| **E2E Test** | `e2e/auth.spec.ts` |

**Preconditions:**
- User is authenticated

**Flow:**
1. User clicks user menu
2. User clicks "Log out"
3. System clears local session state
4. System redirects to DSAccount logout

**Postconditions:**
- `ds_session` cookie is cleared
- User is redirected to DSAccount logout
- Subsequent requests require re-authentication

---

### UC-AUTH-003: Session expires and user re-authenticates

| Attribute | Value |
|-----------|-------|
| **Actor** | User with expired session |
| **Status** | ⚠️ |
| **Requirements** | FR-AUTH-001, FR-AUTH-002 |
| **E2E Test** | `e2e/auth.spec.ts` |

**Preconditions:**
- User was previously authenticated
- Session has expired

**Flow:**
1. User attempts to access a protected resource
2. System detects expired/invalid session
3. System redirects to DSAccount login
4. User re-authenticates
5. System restores user to original destination

**Postconditions:**
- User is authenticated with fresh session
- User is on the page they originally requested

---

## UC-NAV: Navigation Use Cases

### UC-NAV-001: User navigates using header menu

| Attribute | Value |
|-----------|-------|
| **Actor** | Authenticated User |
| **Status** | ⚠️ |
| **Requirements** | FR-NAV-001 |
| **E2E Test** | `e2e/navigation.spec.ts` |

**Preconditions:**
- User is authenticated
- User is on any page

**Flow:**
1. User sees header with logo, nav links, tenant switcher, user menu
2. User clicks a navigation link
3. Page navigates to selected section

**Postconditions:**
- User is on the selected page
- Header remains visible
- Active nav item is highlighted

---

### UC-NAV-002: User switches between DS ecosystem apps

| Attribute | Value |
|-----------|-------|
| **Actor** | Authenticated User |
| **Status** | ⚠️ |
| **Requirements** | FR-NAV-002 |
| **E2E Test** | `e2e/navigation.spec.ts` |

**Preconditions:**
- User is authenticated
- User has access to multiple DS apps

**Flow:**
1. User clicks app-switcher in header
2. System displays available DS apps
3. User selects a different app
4. System navigates to the selected app

**Postconditions:**
- User is on the selected app
- Session is preserved (SSO)

---

### UC-NAV-003: User accesses footer links

| Attribute | Value |
|-----------|-------|
| **Actor** | Any User |
| **Status** | ⚠️ |
| **Requirements** | FR-NAV-003 |
| **E2E Test** | `e2e/navigation.spec.ts` |

**Preconditions:**
- User is on any page

**Flow:**
1. User scrolls to footer
2. User clicks a footer link (Privacy, Terms, etc.)
3. System navigates to selected page

**Postconditions:**
- User is on the selected page

---

### UC-NAV-004: User navigates on mobile device

| Attribute | Value |
|-----------|-------|
| **Actor** | Mobile User |
| **Status** | ⚠️ |
| **Requirements** | FR-NAV-004 |
| **E2E Test** | `e2e/navigation.spec.ts` |

**Preconditions:**
- User is on a mobile device (viewport < 768px)

**Flow:**
1. User sees hamburger menu icon
2. User taps hamburger icon
3. Navigation drawer opens
4. User selects a menu item
5. Drawer closes, page navigates

**Postconditions:**
- User is on the selected page
- Navigation drawer is closed

---

## UC-THEME: Theming Use Cases

### UC-THEME-001: User switches between light and dark theme

| Attribute | Value |
|-----------|-------|
| **Actor** | Any User |
| **Status** | ⚠️ |
| **Requirements** | FR-THEME-001, FR-THEME-003 |
| **E2E Test** | `e2e/theme.spec.ts` |

**Preconditions:**
- User is on any page

**Flow:**
1. User clicks theme toggle in header/settings
2. System switches theme (light ↔ dark)
3. UI updates immediately with new theme

**Postconditions:**
- Theme is applied to entire UI
- CSS variables reflect new theme

---

### UC-THEME-002: User's theme preference persists across sessions

| Attribute | Value |
|-----------|-------|
| **Actor** | Any User |
| **Status** | ⚠️ |
| **Requirements** | FR-THEME-002 |
| **E2E Test** | `e2e/theme.spec.ts` |

**Preconditions:**
- User has previously set a theme preference

**Flow:**
1. User closes browser/clears session
2. User returns to the app
3. System loads saved theme preference
4. UI displays in user's preferred theme

**Postconditions:**
- Theme matches user's last preference
- Preference stored in `ds_prefs` cookie

---

## UC-TENANT: Multi-Tenancy Use Cases

### UC-TENANT-001: User views data scoped to current tenant

| Attribute | Value |
|-----------|-------|
| **Actor** | Authenticated User |
| **Status** | ⚠️ |
| **Requirements** | FR-TENANT-001, FR-TENANT-003 |
| **E2E Test** | `e2e/tenant.spec.ts` |

**Preconditions:**
- User is authenticated
- User belongs to at least one tenant

**Flow:**
1. User navigates to any data page
2. System displays only data for current tenant
3. All API requests include tenant context

**Postconditions:**
- Displayed data is scoped to current tenant
- No cross-tenant data leakage

---

### UC-TENANT-002: User switches between tenants

| Attribute | Value |
|-----------|-------|
| **Actor** | Multi-tenant User |
| **Status** | ⚠️ |
| **Requirements** | FR-TENANT-002, FR-TENANT-004 |
| **E2E Test** | `e2e/tenant.spec.ts` |

**Preconditions:**
- User is authenticated
- User has access to multiple tenants

**Flow:**
1. User clicks tenant switcher in header
2. System displays available tenants
3. User selects a different tenant
4. System updates tenant context
5. Page reloads with new tenant's data

**Postconditions:**
- Current tenant is updated
- All data reflects new tenant context
- `X-Tenant-ID` header updated for API calls

---

## UC-APP: Application-Specific Use Cases

> Add your application's unique use cases below.

<!--
Example structure:

### UC-APP-001: User creates a new project

| Attribute | Value |
|-----------|-------|
| **Actor** | Authenticated User |
| **Status** | ❌ |
| **Requirements** | FR-APP-PROJECT-001 |
| **E2E Test** | `e2e/projects.spec.ts` |

**Preconditions:**
- User is authenticated
- User has permission to create projects

**Flow:**
1. User clicks "New Project" button
2. System displays project creation form
3. User enters project name and description
4. User clicks "Create"
5. System creates project and redirects to project page

**Postconditions:**
- New project exists in database
- User is on the new project's page
- Project appears in project list
-->

---

## E2E Test Coverage Matrix

> Maps use cases to E2E test files and specific test names.

| Use Case | Test File | Test Name(s) |
|----------|-----------|--------------|
| UC-AUTH-001 | `e2e/auth.spec.ts` | `should redirect unauthenticated user to SSO`, `should display user info after SSO callback` |
| UC-AUTH-002 | `e2e/auth.spec.ts` | `should clear session and redirect on logout` |
| UC-AUTH-003 | `e2e/auth.spec.ts` | `should redirect to SSO on expired session` |
| UC-NAV-001 | `e2e/navigation.spec.ts` | `should display header with all required elements` |
| UC-NAV-002 | `e2e/navigation.spec.ts` | `should switch between ecosystem apps` |
| UC-NAV-003 | `e2e/navigation.spec.ts` | `should navigate via footer links` |
| UC-NAV-004 | `e2e/navigation.spec.ts` | `should display mobile navigation drawer` |
| UC-THEME-001 | `e2e/theme.spec.ts` | `should toggle between light and dark theme` |
| UC-THEME-002 | `e2e/theme.spec.ts` | `should persist theme preference` |
| UC-TENANT-001 | `e2e/tenant.spec.ts` | `should display tenant-scoped data` |
| UC-TENANT-002 | `e2e/tenant.spec.ts` | `should switch tenants` |

---

## Linking Use Cases to Requirements

Use cases and requirements serve different purposes:

| Document | Focus | Example |
|----------|-------|---------|
| **USECASES.md** | What users do (scenarios) | "User logs in via SSO" |
| **REQUIREMENTS.md** | How it works (specifications) | "FR-AUTH-001: Users authenticate via DSAccount SSO" |

**Relationship:** Each use case links to one or more requirements. A requirement may support multiple use cases.

```
UC-AUTH-001 (User logs in via SSO)
├── FR-AUTH-001 (SSO authentication)
├── FR-AUTH-002 (Redirect unauthenticated)
└── FR-AUTH-003 (Session includes identity)
```

---

## Adding New Use Cases

1. **Define the use case** with unique ID (UC-{CATEGORY}-{NNN})
2. **Link to requirements** in REQUIREMENTS.md
3. **Create E2E test** with use case ID in describe block
4. **Update coverage matrix** with test references

**E2E Test Pattern:**
```typescript
// e2e/app-feature.spec.ts
test.describe('UC-APP-001: User creates a new project', () => {
  test('should display project creation form', async ({ page }) => {
    // ...
  });
  
  test('should create project and redirect', async ({ page }) => {
    // ...
  });
});
```

---

## Non-Breaking Change Policy

> Use cases are **contracts** with users. Once implemented and tested, they cannot be removed.

1. **New use cases are always additive** — append new IDs
2. **Existing use cases can only be expanded** — add flows, not remove
3. **Deprecated use cases are marked, not deleted:**
   ```markdown
   ❌ DEPRECATED (2026-03): UC-LEGACY-001 - Replaced by UC-AUTH-001
   ```
4. **E2E tests lock use cases** — removing tests requires explicit approval

---

*Template version: 1.0.0*
*Last updated: 2026-03-23*
