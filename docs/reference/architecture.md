# Architecture

> Comprehensive architecture documentation for ds-app-template applications.

---

## Overview

DS Ecosystem apps use a **manifest-based two-layer architecture** that separates replaceable infrastructure from preserved business logic.

### Quick Reference

| Layer | Location | On Update |
|-------|----------|-----------|
| Template | Files in `.template-manifest` | Replaced |
| App | Everything else | Preserved |

---

## System Context

```
┌─────────────────────────────────────────────────────────────┐
│                     CloudFront CDN                          │
│  ┌─────────────────────┐    ┌─────────────────────────┐    │
│  │   Static Assets     │    │    API Gateway          │    │
│  │   (S3 bucket)       │    │    /api/* → Lambda      │    │
│  └─────────────────────┘    └─────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
                              ┌─────────────────┐
                              │  Lambda (Go)    │
                              │  - Handlers     │
                              │  - Middleware   │
                              │  - HAL responses│
                              └─────────────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ▼                   ▼                   ▼
            ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
            │  DynamoDB   │    │  DSAccount  │    │  Secrets    │
            │  (app data) │    │  (SSO/auth) │    │  Manager    │
            └─────────────┘    └─────────────┘    └─────────────┘
```

---

## Key Principles

### 1. Manifest-Based Two-Layer Architecture

Template-owned files are tracked in `.template-manifest`. App-specific code lives outside the manifest.

### 2. HAL/HATEOAS APIs

All API responses follow HAL format with discoverable links. See [api-standards.md](api-standards.md).

### 3. SSO-First Authentication

All apps delegate authentication to DSAccount. No app manages its own user credentials. Exception: DSAccount itself is the identity provider.

### 4. Infrastructure as Code

All infrastructure defined in CDK TypeScript. No manual AWS console changes.

---

## Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| CloudFront | CDN, HTTPS termination, routing |
| S3 | Static frontend assets |
| API Gateway | HTTP routing, CORS |
| Lambda | Business logic, API handlers |
| DynamoDB | Application data |
| DSAccount | Authentication, session management |
| Secrets Manager | Sensitive configuration |

---

## Two-Layer Architecture

Ownership is defined by `.template-manifest`, not physical directory separation. The manifest lists paths that are **template-owned**—everything else is **app-owned**.

When you run `update-from-template.sh`:
1. Reads `.template-manifest` to identify template-owned paths
2. Copies those paths from ds-app-template
3. Leaves app-owned paths completely untouched
4. Updates `.template-version`

### Template Layer (Manifest-Owned)

**Ownership:** ds-app-template repository  
**On update:** Replaced by `update-from-template.sh`  
**Rule:** Never modify directly in derived apps

From `.template-manifest`:
```
# Build & CI
.github/workflows/

# Documentation
docs/

# Infrastructure (CDK)
infra/

# Scripts
scripts/

# Backend: shared infrastructure
backend/internal/buildinfo/
backend/internal/discovery/
backend/internal/hal/
backend/internal/health/
backend/internal/middleware/
backend/internal/session/
backend/internal/auth/
backend/cmd/

# Frontend: shared components and utilities
frontend/public/
frontend/src/components/
frontend/src/hooks/
frontend/package.json
frontend/vite.config.ts
frontend/tsconfig.json
frontend/tsconfig.node.json
frontend/tailwind.config.js
frontend/postcss.config.js

# Root config files
.gitignore
.npmrc
pnpm-workspace.yaml

# Template metadata
.template-version
.template-manifest
```

### App Layer (App-Owned)

**Ownership:** Your app repository  
**On update:** Never touched  
**Rule:** All business logic goes here

```
frontend/src/app/              # Your frontend code
├── pages/                     # Route components
│   └── HomePage.tsx
├── features/                  # Feature modules
│   └── [feature]/
│       ├── components/
│       ├── hooks/
│       └── types.ts
├── assets/                    # App-specific assets
├── config.ts                  # App configuration
├── Layout.tsx                 # Layout using DSAppShell
└── index.ts                   # Exports

backend/internal/handlers/     # Your API handlers
└── *.go                       # Domain-specific endpoints
```

---

## Directory Structure

### Backend

```
backend/
├── cmd/
│   └── api/
│       └── main.go           # [TEMPLATE] Entry point, route setup
├── internal/
│   ├── auth/                 # [TEMPLATE] SSO middleware
│   │   ├── handlers.go       # Auth endpoints
│   │   └── middleware.go     # Auth middleware
│   ├── buildinfo/            # [TEMPLATE] Build metadata
│   │   └── buildinfo.go
│   ├── discovery/            # [TEMPLATE] HAL root
│   │   └── discovery.go
│   ├── hal/                  # [TEMPLATE] HAL helpers
│   │   └── hal.go
│   ├── handlers/             # [APP] Your handlers
│   │   └── *.go
│   ├── health/               # [TEMPLATE] Health check
│   │   └── handler.go
│   ├── middleware/           # [TEMPLATE] Common middleware
│   │   ├── correlation.go
│   │   ├── logging.go
│   │   └── recovery.go
│   └── session/              # [TEMPLATE] Session mgmt
│       ├── middleware.go
│       └── session.go
└── go.mod
```

### Frontend

```
frontend/
├── public/                  # [TEMPLATE] Static assets (logos, favicon)
└── src/
    ├── app/                 # [APP] Your pages, features, config
    └── (everything else)    # [TEMPLATE] Shell, components, api utils
```

---

## Decision Guide: Where Does It Go?

### Use Template Layer When:

| Scenario | Example |
|----------|---------|
| Consistent across all DS apps | Auth middleware, health checks |
| Infrastructure pattern | CDK constructs, Lambda setup |
| Common middleware | Logging, CORS, correlation ID |
| Build tooling | Vite config, build scripts |
| UI primitives | LoadingSpinner, ErrorBoundary |

### Use App Layer When:

| Scenario | Example |
|----------|---------|
| Business logic | User registration, billing calculations |
| Domain models | Project, Issue, User (for that app) |
| Feature code | Dashboard specific to this app |
| API handlers | Endpoints unique to this app |
| Route definitions | This app's pages and navigation |
| App configuration | Name, logo, feature flags |

### Use Shared Packages When:

| Scenario | Package | Example |
|----------|---------|---------|
| Auth/session hooks | `@digistratum/ds-core` | `useAuth`, `useTheme` |
| Layout components | `@digistratum/layout` | `DSAppShell`, `GdprBanner` |
| Design tokens | `@digistratum/design-tokens` | Colors, typography |

---

## Import Hierarchy

```typescript
// 1. External packages
import React from 'react';

// 2. Shared DS packages (from ds-app-resources)
import { useAuth, DS_URLS } from '@digistratum/ds-core';
import { DSAppShell } from '@digistratum/layout';

// 3. Template layer (local, replaceable)
import { LoadingSpinner } from '../components/LoadingSpinner';
import { useHALNavigation } from '../hooks/useHALNavigation';

// 4. App layer (local, preserved)
import { HomePage } from './app/pages';
import config from './app/config';
```

---

## Extension Points

When template components need customization:

### Props-Based Customization

The app's `Layout.tsx` wraps `DSAppShell` with app-specific props:

```tsx
// frontend/src/app/Layout.tsx
import { DSAppShell } from '@digistratum/layout';
import config from './config';

export function Layout({ children }: LayoutProps) {
  return (
    <DSAppShell
      appName={config.name}
      appLogo={config.logo}
      currentAppId={config.id}
      showAppSwitcher={true}
    >
      {children}
    </DSAppShell>
  );
}
```

### Configuration-Based Customization

App-specific values in `config.ts`:

```typescript
// frontend/src/app/config.ts
const config = {
  id: 'my-app',
  name: 'My Application',
  logo: '/logo.svg',
  features: {
    darkMode: true,
    gdprBanner: true,
  },
};

export default config;
```

---

## Ownership Model

Every file in a DS app belongs to exactly one owner:

| Owner | Location | On Template Update |
|-------|----------|-------------------|
| **Template** | Listed in `.template-manifest` | **Replaced** |
| **App** | Not in manifest | **Preserved** |

### Verification

```bash
go run ./tools/cmd/check-manifest

# CI runs this with --strict (fails on unknown files)
go run ./tools/cmd/check-manifest --strict
```

### Adding New Files

**Template feature (shared across apps):**
1. Add file to appropriate template location
2. Add path to `.template-manifest`
3. Run `check-manifest` to verify

**App feature (specific to this app):**
1. Add file to `backend/internal/handlers/` or `frontend/src/app/`
2. Do NOT add to manifest
3. Run `check-manifest` to verify

### Infrastructure Extension

For app-specific CDK resources:

```
infra/lib/app/              # APP-OWNED
└── resources.ts            # Your tables, queues, etc.
```

Import in the main stack:

```typescript
// infra/lib/app-stack.ts (template)
import { AppResources } from './app/resources';
new AppResources(this, 'AppResources');
```

---

## Anti-Patterns and Fixes

### ❌ Modifying Template-Owned Files

**Problem:** Editing files like `frontend/src/components/LoadingSpinner.tsx` in your app.

**Fix:** These will be overwritten on next template update. Submit PR to ds-app-template instead.

### ❌ Business Logic in Template Paths

**Problem:** Adding app-specific handlers to `backend/internal/middleware/`.

**Fix:** Move to `backend/internal/handlers/`. Only `handlers/` is app-owned.

### ❌ Copying Template Code to App Layer

**Problem:** Copying `components/` into `app/` to customize.

**Fix:** Use props/configuration, or request extension point upstream.

### ❌ Ignoring the Manifest

**Problem:** Adding new template-owned paths without updating `.template-manifest`.

**Fix:** Update manifest in ds-app-template when adding new template paths.

---

## Related Documents

- [Tech Stack](tech-stack.md) — Technologies used
- [Architecture Decisions](decisions/README.md) — ADRs
- [API Standards](api-standards.md) — HAL/HATEOAS patterns
