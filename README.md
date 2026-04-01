# ds-noc-v2

> __APP_DESCRIPTION__

<!-- Delete this section after creating your app -->
## About This Template

This is the DS Ecosystem app template. After running `create-app`, the tokens above will be replaced with your app's name and description.

**Template features:**
- Full-stack boilerplate – React frontend + Go backend + AWS infrastructure
- Two-layer architecture – Infrastructure layer updates without touching your business logic
- CI/CD pipeline – GitHub Actions with OIDC authentication to AWS
- DS integration – SSO via DSAccount, shared packages from ds-app-resources
- Scaffolding tools – Generate endpoints, models, pages, and full feature slices
<!-- End delete section -->

## Prerequisites

- **Node.js** 20+ and **pnpm** 8+
- **Go** 1.24+
- **lefthook** (pre-commit hooks) – `brew install lefthook`
- **AWS CLI** configured with credentials
- **GitHub CLI** (`gh`) authenticated
- AWS OIDC provider configured for GitHub Actions
- **IAM permissions** for ecosystem registry S3 access (see [CI_CD.md](./docs/CI_CD.md))

## Quick Start

### Create Your App

```bash
# From the template directory
cd ~/repos/digistratum/ds-app-template

# Create a new app with GitHub repo
go run ./tools/cmd/create-app -n workforce --github
```

This creates `ds-app-workforce/`, initializes git, creates the `DigiStratum/ds-app-workforce` GitHub repo, and reports required secrets.

**Naming Convention:**

| APP_NAME | Repo | Dev Stack | Prod Stack | Dev Domain | Prod Domain |
|----------|------|-----------|------------|------------|-------------|
| workforce | ds-app-workforce | ds-app-workforce-dev | ds-app-workforce-prod | workforce.dev.digistratum.com | workforce.digistratum.com |

Domains are derived at deploy time from the ecosystem registry—no hardcoding needed.

### Develop Locally

```bash
cd ~/repos/digistratum/myapp

# Install dependencies
pnpm install

# Install pre-commit hooks
lefthook install

# Start frontend (http://localhost:5173)
pnpm dev

# In separate terminal: start backend
cd backend && go run cmd/api/main.go

# Run tests
pnpm test
cd backend && go test ./...
```

### Deploy

Push to `main` triggers automatic deployment via GitHub Actions.

**Required GitHub Secrets:**

| Secret | Description |
|--------|-------------|
| `AWS_ROLE_ARN` | IAM role ARN for OIDC authentication |
| `AWS_ACCOUNT_ID` | 12-digit AWS account ID |
| `E2E_TEST_USER_EMAIL` | Test user email for E2E tests |
| `E2E_TEST_USER_PASSWORD` | Test user password for E2E tests |

*Note:* E2E secrets are required for post-deploy validation. Alternatively, use `E2E_SESSION_TOKEN` for pre-authenticated tests.

## Architecture Overview

Apps use a **two-layer architecture** that separates template infrastructure from your business logic:

| Layer | Path | On Template Update | Contains |
|-------|------|-------------------|----------|
| **Template** | `template/` | Replaced | Shell, middleware, infra, common components |
| **App** | `app/` | Preserved | Pages, features, handlers, domain logic |

**Rule:** Never modify `template/` directly. Improvements go upstream to ds-app-template.

See [docs/reference/architecture.md](./docs/reference/architecture.md) for the full overview and directory structure.

## Keeping Up to Date

```bash
# Preview what would change
go run ./tools/cmd/update-from-template --dry-run ~/repos/digistratum/myapp

# Apply template updates
go run ./tools/cmd/update-from-template ~/repos/digistratum/myapp
```

The update tool replaces the template layer while preserving your app layer. See [docs/updating-apps.md](./docs/updating-apps.md).

## Pre-Commit Hooks

This template uses [lefthook](https://github.com/evilmartians/lefthook) for pre-commit hook management.

### Installation

```bash
# Install lefthook (if not already installed)
brew install lefthook

# Enable hooks in your repo
lefthook install
```

### Hook Chain

Pre-commit runs the following checks in order:

| Check | What It Does | Path |
|-------|-------------|------|
| go fmt | Format Go code | backend/ |
| go vet | Lint Go code | backend/ |
| go test -short | Run short Go unit tests | backend/ |
| prettier | Format JS/TS/CSS/JSON/MD | frontend/ |
| eslint | Lint JS/TS code | frontend/ |
| npm test | Run frontend unit tests | frontend/ |

### Skipping Hooks

Use `--no-verify` to skip hooks when necessary (escape hatch):

```bash
git commit --no-verify -m "WIP: emergency fix"
```

### Configuration

See `lefthook.yml` at the repo root for hook configuration.

## Learn More

### Documentation

| Doc | Description |
|-----|-------------|
| [Creating a New App](./docs/creating-new-app.md) | Step-by-step guide to spawn a new app |
| [Updating Apps](./docs/updating-apps.md) | Apply template updates to existing apps |
| [Maturity Model](./docs/MATURITY.md) | Application quality levels and checklists |
| [Maturity Schema](./docs/maturity-schema.md) | Machine-readable maturity.yaml reference |
| [Architecture](./docs/reference/architecture.md) | System architecture overview |
| [Tech Stack](./docs/reference/tech-stack.md) | Technology choices and rationale |
| [Testing](./docs/reference/testing.md) | Test strategies and coverage |
| [Deployment](./docs/runbooks/deployment.md) | CI/CD and AWS configuration |
| [Troubleshooting](./docs/runbooks/troubleshooting.md) | Common issues and solutions |

### Maturity Assessment

Check your application's maturity level during development:

```bash
# Check current maturity level
make -C tools maturity

# Output as JSON (for CI integration)
make -C tools maturity-json

# See gaps to next level
make -C tools maturity-gaps
```

See [Maturity Model](./docs/MATURITY.md) for level definitions and requirements.

### Scaffolding Tools

All tools are Go programs in `tools/cmd/`. Run via `go run ./tools/cmd/<tool>`:

| Tool | Purpose |
|------|---------|
| `create-app` | Create new app from template |
| `update-from-template` | Update app with latest template |
| `add-endpoint` | Scaffold API handler + route + HAL link |
| `add-model` | Scaffold DynamoDB model + repository |
| `add-service` | Scaffold business logic service |
| `add-page` | Scaffold React page + route |
| `add-component` | Scaffold React component + test |
| `add-hook` | Scaffold React hook + test |
| `add-feature` | Full vertical slice (endpoint + model + page) |

See [AGENTS.md](./AGENTS.md) for detailed tool usage.

### Shared Packages

Apps consume packages from [ds-app-resources](https://github.com/DigiStratum/ds-app-resources):

| Package | Purpose |
|---------|---------|
| `@digistratum/layout` | AppShell, Header, Footer, navigation |
| `@digistratum/ds-core` | Auth hooks, API client, utilities |
| `@digistratum/design-tokens` | Colors, typography, spacing |

```typescript
import { useAuth, useSessionData } from '@digistratum/ds-core';
import { AppShell, Header } from '@digistratum/layout';
```

### Related Repositories

- [ds-app-resources](https://github.com/DigiStratum/ds-app-resources) – Shared packages
- [DSAccount](https://github.com/DigiStratum/DSAccount) – Authentication & SSO
- [DSKanban](https://github.com/DigiStratum/DSKanban) – Project management
- [GoTools](https://github.com/DigiStratum/GoTools) – Scaffolding tool library

## Version

Current template version: see `.template-version`

Apps track which template version they're based on via their own `.template-version` file.
