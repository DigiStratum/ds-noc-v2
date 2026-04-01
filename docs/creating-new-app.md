# Creating a New App

Step-by-step guide to creating a new DS Ecosystem app from the template.

## Prerequisites

Before creating an app, ensure you have:

- **Node.js** 20+ and **pnpm** 8+
- **Go** 1.24+
- **AWS CLI** configured with appropriate permissions
- **GitHub CLI** (`gh`) authenticated
- AWS account with OIDC provider configured for GitHub Actions

## Quick Start

```bash
cd ~/repos/digistratum/ds-app-template
go run ./tools/cmd/create-app -n workforce --github
```

This creates `ds-app-workforce/` directory and `DigiStratum/ds-app-workforce` GitHub repo.

## Naming Conventions

Apps follow a consistent naming pattern. You provide `APP_NAME`, everything else is derived:

| APP_NAME | Repo | Dev Stack | Prod Stack | Dev Domain | Prod Domain |
|----------|------|-----------|------------|------------|-------------|
| workforce | ds-app-workforce | ds-app-workforce-dev | ds-app-workforce-prod | workforce.dev.digistratum.com | workforce.digistratum.com |
| kanban | ds-app-kanban | ds-app-kanban-dev | ds-app-kanban-prod | kanban.dev.digistratum.com | kanban.digistratum.com |
| marketplace | ds-app-marketplace | ds-app-marketplace-dev | ds-app-marketplace-prod | marketplace.dev.digistratum.com | marketplace.digistratum.com |

**Key points:**
- **Repo name:** Always `ds-app-{APP_NAME}`
- **Stack name:** `{REPO_NAME}-{env}` (e.g., `ds-app-workforce-dev`)
- **Domain:** Computed at deploy time from ecosystem registry (no hardcoding)
- **Multi-ecosystem:** Same app, different domains (e.g., `workforce.leapkick.com`)

## Step-by-Step Process

### 1. Clone the Template Repository

```bash
git clone https://github.com/DigiStratum/ds-app-template.git
cd ds-app-template
```

### 2. Run create-app

The tool can run interactively or with flags:

**Interactive:**
```bash
go run ./tools/cmd/create-app
```

**Non-interactive:**
```bash
go run ./tools/cmd/create-app \
  -n marketplace \
  -N "DS Marketplace" \
  --github \
  -y
```

### 3. Configuration Prompts

| Prompt | Description | Example |
|--------|-------------|---------|
| App name | Lowercase identifier (no hyphens unless needed) | `workforce` |
| Display name | Human-readable name | `DS Workforce` |

Note: Domain, AWS account, and region are **no longer prompted**—they come from the ecosystem registry at deploy time.

### 4. What Gets Created

The tool copies the template and replaces placeholders. Example for `workforce`:

```
~/repos/digistratum/ds-app-workforce/
├── .github/                 # GitHub Actions workflows
│   └── workflows/
├── .template-manifest       # Defines template-owned vs app-owned files
├── .template-version        # Template version marker
├── backend/                 # Go backend
│   ├── cmd/api/             # Lambda entry point
│   ├── internal/
│   │   ├── auth/            # SSO/session middleware
│   │   ├── discovery/       # HAL API discovery
│   │   ├── hal/             # HAL response helpers
│   │   ├── handlers/        # [APP] Your API handlers
│   │   ├── health/          # Health check endpoint
│   │   └── middleware/      # Request middleware
│   ├── go.mod
│   └── go.sum
├── frontend/                # React frontend
│   ├── e2e/                 # Playwright E2E tests
│   ├── src/
│   │   ├── api/             # API client utilities
│   │   ├── app/             # [APP] Your pages, features, config
│   │   ├── components/      # Shared UI components
│   │   └── hooks/           # React hooks
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   └── tailwind.config.js
├── infra/                   # AWS CDK infrastructure
│   ├── bin/                 # CDK app entry
│   ├── lib/                 # Stack definitions
│   ├── cdk.json
│   └── package.json
├── docs/                    # Documentation
│   ├── reference/
│   │   ├── architecture.md  # Architecture overview
│   │   └── tech-stack.md    # Technologies used
│   └── REQUIREMENTS.md      # Requirements template
├── tools/                   # Go-based dev tools
│   ├── update-from-template.sh  # Pull template updates
│   └── add-endpoint.sh      # Scaffold new API endpoints
├── AGENTS.md                # AI agent context
├── README.md                # App-specific README
├── package.json             # Root package.json
└── pnpm-workspace.yaml      # pnpm workspace config
```

**App-owned locations** (preserved on template updates):
- `backend/internal/handlers/` — Your API handlers
- `frontend/src/app/` — Your pages, features, config
- `README.md` — Your app description

### 5. GitHub Repository Setup

If `--github` was specified, the script:
1. Creates a new repo in the DigiStratum organization
2. Sets up branch protection
3. Pushes initial commit
4. **Does NOT set secrets** – see below

### 6. Configure Secrets

Manually add these secrets in GitHub → Settings → Secrets:

| Secret | Value | Required |
|--------|-------|----------|
| `AWS_ROLE_ARN` | IAM role ARN for OIDC | Yes |
| `AWS_ACCOUNT_ID` | 12-digit account ID | Yes |
| `NPM_TOKEN` | npm access token (if private packages) | No |

### 7. First Deployment

Push to `develop` triggers automatic deployment to dev environment:

```bash
cd ~/repos/digistratum/ds-app-workforce
git push -u origin develop
```

Monitor the Actions tab for deployment status.

## Command-Line Options

| Option | Short | Description |
|--------|-------|-------------|
| `--name` | `-n` | App name (e.g., `workforce`) |
| `--display-name` | `-N` | Display name (e.g., `DS Workforce`) |
| `--github` | `-g` | Set up GitHub remote |
| `--org` | `-o` | GitHub organization (default: DigiStratum) |
| `--yes` | `-y` | Non-interactive mode |
| `--no-git` | | Skip git init (for refreshing existing repos) |
| `--help` | `-h` | Show help |

## Placeholders Replaced

The script replaces these tokens in all files:

| Token | Replaced With | Example |
|-------|--------------|---------|
| `ds-noc-v2` | Short app name | `workforce` |
| `__REPO_NAME__` | Full repository name | `ds-app-workforce` |
| `DS Noc V2` | Display name | `Workforce` |
| `noc-v2` | Subdomain (same as APP_NAME) | `workforce` |
| `dsnocv2` | App identifier for registry (no hyphens) | `workforce` |
| `DigiStratum` | GitHub organization | `DigiStratum` |

**Deprecated tokens (removed):**
- `noc-v2.digistratum.com` - Now computed at runtime from subdomain + ecosystem
- `171949636152` - Now from ecosystem registry, not hardcoded
- `us-west-2` - Now from ecosystem registry, not hardcoded

## Template Configuration File

The `.template-config` file stores your app's placeholder values in KEY=VALUE format:

```bash
# .template-config
APP_NAME=workforce
REPO_NAME=ds-app-workforce
APP_DISPLAY_NAME=Workforce
APP_SUBDOMAIN=workforce
APP_ID=workforce
GITHUB_ORG=DigiStratum
SUPPORT_EMAIL=info@digistratum.com
APP_DESCRIPTION=Workforce management system
```

This file is:
- **Created automatically** by `create-app`
- **Read by** `update-from-template` for placeholder substitution
- **Read by** `substitute-tokens` (GoTools) for standalone substitution

If you need to change your app's configuration, edit this file and re-run `update-from-template`.

**Note:** AWS account/region and domain are NOT stored here - they come from the ecosystem registry at deploy time.

## Post-Creation Checklist

- [ ] Verify GitHub repository created
- [ ] Add required secrets (AWS_ROLE_ARN, AWS_ACCOUNT_ID)
- [ ] Push initial commit
- [ ] Verify CI/CD pipeline succeeds
- [ ] Access deployed app at domain

## Troubleshooting

### "AWS account ID must be 12 digits"
Ensure account ID is exactly 12 numeric characters.

### "Failed to create GitHub repository"
Check `gh auth status` and ensure you have org write access.

### CDK deployment fails
Verify AWS OIDC provider is configured for the repository.

See [runbooks/deployment.md](runbooks/deployment.md) for deployment troubleshooting.
