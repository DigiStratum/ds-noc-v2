# Getting Started

> Quick start guide for developing with ds-app-template.

---

## Prerequisites

| Tool | Version | Check |
|------|---------|-------|
| Go | 1.21+ | `go version` |
| Node.js | 20+ | `node --version` |
| pnpm | 8+ | `pnpm --version` |
| AWS CLI | 2.x | `aws --version` |

---

## Create New App

```bash
# Clone template
git clone https://github.com/DigiStratum/ds-app-template.git my-app
cd my-app

# Run setup script
go run ./tools/cmd/create-app my-app

# This will:
# - Rename the app
# - Set up GitHub repo
# - Configure AWS secrets
# - Create GitHub environments
```

---

## Local Development

### Backend (Go)

```bash
cd backend

# Install dependencies
go mod download

# Run tests
go test ./...

# Run locally (port 8080)
go run cmd/api/main.go
```

### Frontend (React)

```bash
cd frontend

# Install dependencies
pnpm install

# Run dev server (port 5173)
pnpm dev

# Run tests
pnpm test
```

---

## Project Structure

```
my-app/
├── backend/
│   ├── cmd/api/             # Lambda entry point
│   └── internal/
│       ├── handlers/        # [APP] Your API handlers
│       └── ...              # Auth, HAL, middleware (template)
│
├── frontend/
│   ├── e2e/                 # Playwright E2E tests
│   └── src/
│       ├── app/             # [APP] Your pages, features, config
│       ├── api/             # API client utilities
│       ├── components/      # Shared UI components
│       └── hooks/           # React hooks
│
├── infra/                   # CDK infrastructure
│   ├── bin/                 # CDK app entry
│   └── lib/                 # Stack definitions
│
├── docs/                    # Documentation
│   ├── reference/
│   │   ├── architecture.md  # System design
│   │   └── tech-stack.md    # Technologies used
│   └── REQUIREMENTS.md      # Requirements
│
├── tools/                   # Go-based dev tools
├── .template-manifest       # Template-owned files list
├── .template-version        # Current template version
├── AGENTS.md                # AI agent context
└── README.md                # App description
```

---

## Add a New Endpoint

```bash
# Use the scaffold script
./go run ./tools/cmd/add-endpoint POST /api/items CreateItem

# This creates:
# - backend/internal/handlers/items.go (stub)
# - backend/internal/handlers/items_test.go (test stub)
# - Registers route in main.go
# - Adds to discovery.go
```

---

## Deploy

Push to `main` branch triggers CI/CD:

1. **Build & Test** — All checks must pass
2. **Deploy to Dev** — Automatic on push
3. **Deploy to Prod** — Manual workflow dispatch

See [runbooks/deployment.md](runbooks/deployment.md) for details.

---

## Next Steps

1. Read [Architecture Overview](reference/architecture.md)
2. Review [Code Conventions](reference/conventions.md)
3. Understand [API Standards](reference/api-standards.md)
4. Check [Maturity Model](MATURITY.md) for quality targets
