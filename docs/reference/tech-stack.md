# DS App Tech Stack

**This file is template-maintained.** Do not edit directly — changes will be overwritten by template updates.

## Overview

All DS apps share a consistent technology stack for predictable development, deployment, and maintenance.

## Stack Summary

| Layer | Technology | Version/Notes |
|-------|------------|---------------|
| Frontend | React, TypeScript, Vite, TailwindCSS | React 18+ |
| Backend | Go, net/http (stdlib) | Go 1.21+ |
| Infrastructure | AWS CDK (TypeScript) | CloudFront, Lambda, DynamoDB, S3 |
| CI/CD | GitHub Actions | OIDC authentication |
| Package Registry | packages.digistratum.com | S3-hosted tarballs |

## Frontend Stack

### Core
- **React 18** — Functional components with hooks
- **TypeScript** — Strict mode enabled
- **Vite** — Build tooling and dev server

### Styling
- **TailwindCSS** — Utility-first CSS
- **@digistratum/layout** — Shared shell/chrome components

### Testing
- **Vitest** — Unit and component tests
- **Playwright** — E2E tests

## Backend Stack

### Core
- **Go 1.21+** — Standard library HTTP server
- **net/http** — No external routing frameworks

### Patterns
- **HAL/HATEOAS** — All API responses follow HAL format
- **Structured logging** — JSON format for all operations

### Storage
- **DynamoDB** — Primary data store (production)
- **SQLite** — Local development alternative

## Infrastructure Stack

### AWS Services
- **CloudFront** — CDN and HTTPS termination
- **Lambda** — Backend compute (ARM64)
- **DynamoDB** — NoSQL data storage
- **S3** — Static assets and frontend hosting
- **Secrets Manager** — Secret storage (no secrets in code)

### Deployment
- **AWS CDK** — Infrastructure as code (TypeScript)
- **GitHub Actions** — CI/CD pipelines
- **OIDC** — Secure AWS authentication (no long-lived credentials)

## Package Registry

Internal packages hosted at `packages.digistratum.com`:
- S3-hosted tarballs
- Versioned releases
- Used by both frontend (npm) and backend (Go modules) where applicable

## Version Requirements

| Component | Minimum Version |
|-----------|-----------------|
| Go | 1.21 |
| Node.js | 18 LTS |
| React | 18 |
| AWS CDK | 2.x |

## Related Documentation

- [Architecture](architecture.md) — System design and layer architecture
- [Architecture Decisions](decisions/README.md) — ADRs
