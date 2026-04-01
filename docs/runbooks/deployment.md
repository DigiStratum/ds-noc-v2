# Deployment Guide

This document covers CI/CD pipeline configuration and environment setup for DS apps.

---

## CI/CD Pipeline

### Overview

The DS Ecosystem App template includes a GitHub Actions CI/CD pipeline that:
- Validates code on every PR
- Builds and tests on pushes to protected branches
- Deploys automatically based on branch:
  - `develop` → dev environment
  - `release/*` → staging environment
  - `main` → production environment

### Required Setup

#### GitHub Secrets

| Secret | Description | How to Get |
|--------|-------------|------------|
| `AWS_ROLE_ARN` | IAM role ARN for OIDC | From AWS CDK bootstrap or manual setup |
| `AWS_ACCOUNT_ID` | Target AWS account number | AWS Console → Account ID |
| `NPM_TOKEN` | (Optional) For private npm packages | npm.js → Access Tokens |

#### GitHub Environments

Create these environments in repo Settings → Environments:

| Environment | Branch Pattern | Protection |
|-------------|----------------|------------|
| `development` | `develop` | None |
| `staging` | `release/*` | Optional reviewers |
| `production` | `main` | Required reviewers, wait timer |

#### AWS OIDC Setup

The pipeline uses OIDC (no long-lived credentials). Ensure your AWS account has:

1. OIDC Identity Provider for GitHub Actions
2. IAM Role with trust policy for your repo
3. Permissions for CDK, Lambda, CloudFront, S3, DynamoDB

See [AWS OIDC Setup Guide](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/configuring-openid-connect-in-amazon-web-services)

### Pipeline Stages

#### 1. Changes Detection

Uses `dorny/paths-filter` to determine what changed:
- Backend: `backend/`, `app/handlers/`, `app/domain/`
- Frontend: `frontend/`, `template/components/`, `app/pages/`
- Infra: `infra/`, `template/infra/`

Only affected jobs run, saving CI minutes.

#### 2. Build & Test

| Component | Tests | Build Output |
|-----------|-------|--------------|
| Backend | `go test -race` | `backend/dist/bootstrap` (arm64) |
| Frontend | `vitest` | `frontend/dist/` |
| Infra | TypeScript compile | CDK synth at deploy time |

#### 3. Deploy

Deploys use CDK with environment context:
```bash
cdk deploy APP-env --context env=dev|staging|prod
```

#### 4. Post-Deploy Validation

After each deployment:
1. **CloudFormation Status**: Verify stack isn't in ROLLBACK/FAILED state
2. **Frontend Assets**: Fetch index.html, verify JS/CSS bundles load
3. **Health Check**: Hit `/api/health` endpoint, expect `{"status":"healthy"}`

---

## Environments

### Environment Summary

| Environment | Domain Pattern | Branch | Deployment |
|-------------|----------------|--------|------------|
| Dev | `*.dev.digistratum.com` | `main` | Automatic on merge |
| Prod | `*.digistratum.com` | `main` | Manual promotion |

### Domain Structure

#### Development
```
myapp.dev.digistratum.com
  ├── / (frontend - CloudFront → S3)
  └── /api/* (backend - CloudFront → Lambda)
```

#### Production
```
myapp.digistratum.com
  ├── / (frontend - CloudFront → S3)
  └── /api/* (backend - CloudFront → Lambda)
```

### Deployment Flow

```
PR Created
    │
    ▼
CI Validation (lint, test, build)
    │
    ▼
PR Merged to main
    │
    ▼
Auto-deploy to Dev ──────────────────┐
    │                                │
    ▼                                ▼
Dev Environment             CDK diff check
    │                                │
    ▼                                │
Manual Promotion Workflow ◄──────────┘
    │
    ▼
Production Environment
```

### AWS Resources by Environment

| Resource | Dev | Prod |
|----------|-----|------|
| Lambda | `myapp-dev-api` | `myapp-prod-api` |
| DynamoDB | `myapp-dev-*` | `myapp-prod-*` |
| S3 (frontend) | `myapp-dev-frontend` | `myapp-prod-frontend` |
| CloudFront | Separate distribution | Separate distribution |
| Secrets | `myapp/dev/*` | `myapp/prod/*` |

### Environment Variables

Set via Lambda environment, not CDK context:

| Variable | Description |
|----------|-------------|
| `ENVIRONMENT` | `dev` or `prod` |
| `LOG_LEVEL` | `debug` (dev), `info` (prod) |
| `CORS_ORIGIN` | Environment-specific domain |
| `TXNLOG_GROUP` | CloudWatch log group for ecosystem transaction logs (e.g., `/ds/ecosystem/transactions`) |
| `APP_ID` | Application identifier for txnlog (e.g., `myapp`) |
| `APP_NAME` | Human-readable app name for txnlog (e.g., `My App`) |
| `ENV` | Environment name for txnlog (e.g., `prod`, `dev`) |

**Note:** The `TXNLOG_*` variables enable centralized ecosystem transaction logging. If `TXNLOG_GROUP` is not set, the middleware gracefully falls back to local logging (no crash).

### Secrets Management

Secrets stored in AWS Secrets Manager:
- `myapp/dev/db-credentials`
- `myapp/prod/db-credentials`

Referenced by ARN in CDK, never hardcoded.

### Health Checks

#### Endpoint
```
GET /api/health
```

#### Response
```json
{
  "status": "healthy",
  "version": "1.2.3",
  "environment": "dev"
}
```

#### Monitoring
- CloudWatch alarm on health check failures
- 5xx error rate monitoring
- Lambda duration/timeout alerts

---

## Failure Modes & Remediation

### Build Failures

| Symptom | Likely Cause | Remediation |
|---------|--------------|-------------|
| `go mod download` fails | Missing dependencies, network issue | Check `go.mod`, retry |
| `pnpm install` fails | Lock file mismatch, missing packages | Run `pnpm install` locally, commit lock file |
| TypeScript errors | Type mismatches | Fix types, run `pnpm typecheck` locally |
| Lint failures | Code style violations | Run `pnpm lint --fix` locally |

### Test Failures

| Symptom | Likely Cause | Remediation |
|---------|--------------|-------------|
| Go tests fail | Logic error, race condition | Run locally with `-race`, fix code |
| Frontend tests fail | Component/hook issue | Run `pnpm test` locally, debug |
| Flaky tests | Timing issues, external dependencies | Add retries, mock externals |

### Deploy Failures

| Symptom | Likely Cause | Remediation |
|---------|--------------|-------------|
| OIDC auth fails | Trust policy mismatch | Check IAM role trust policy includes repo/branch |
| CDK synth fails | Missing context, invalid config | Run `cdk synth` locally with same context |
| CloudFormation ROLLBACK | Resource conflict, quota | Check CF events, fix resource/increase quota |
| Lambda fails to start | Binary incompatible, missing env vars | Check build is linux/arm64, verify env vars |

### Health Check Failures

| Symptom | Likely Cause | Remediation |
|---------|--------------|-------------|
| 000 (timeout) | Lambda cold start, DNS not propagated | Wait, retry; check CloudWatch logs |
| 403 | CloudFront auth issue, wrong origin | Check CF origin config, Lambda URL |
| 500 | Lambda error | Check CloudWatch logs for Lambda |
| 502/504 | Lambda timeout, API Gateway issue | Increase Lambda timeout, check APIGW |
| Health returns but no `{"status":"healthy"}` | Handler wrong response | Fix `HealthHandler` |

### CloudFront/DNS Issues

| Symptom | Likely Cause | Remediation |
|---------|--------------|-------------|
| SSL cert error | ACM cert not issued | Check ACM, ensure DNS validation complete |
| Site not accessible | DNS not propagated | Wait 5-10 min, check Route53 |
| Stale content | CloudFront cache | Create invalidation: `/*` |
| Assets 404 | S3 sync incomplete | Check S3 bucket, verify deploy uploaded files |

---

## Manual Interventions

### Force Redeploy

Use workflow_dispatch in GitHub Actions UI, or:
```bash
gh workflow run ci-cd.yml -f environment=prod
```

### Rollback

CDK doesn't have built-in rollback. Options:
1. Revert commit, push to trigger redeploy
2. Use CloudFormation console to roll back stack
3. Deploy previous commit: `git checkout <commit> && gh workflow run ci-cd.yml`

#### Lambda Rollback
```bash
# Rollback Lambda to previous version
aws lambda update-alias --function-name myapp-prod-api \
  --name live --function-version $PREVIOUS_VERSION
```

### Cache Invalidation

```bash
aws cloudfront create-invalidation \
  --distribution-id E1234567890 \
  --paths "/*"
```

### Check Lambda Logs

```bash
aws logs tail /aws/lambda/APP-prod-api --follow
```

---

## Monitoring

After deployment, monitor:
- CloudWatch Metrics: Lambda invocations, errors, duration
- CloudWatch Logs: Lambda logs, API Gateway access logs
- CloudFront: Cache hit ratio, error rate

Set up CloudWatch Alarms for:
- Lambda errors > 5 in 5 minutes
- API latency p99 > 3 seconds
- 5xx error rate > 1%

---

## Access Control

| Environment | Who |
|-------------|-----|
| Dev | All team members |
| Prod | Deploy via GitHub Actions only |

AWS console access requires MFA. No direct production deployments.

---

## Local Testing

Before pushing, validate locally:

```bash
# Backend
cd backend && go test -race ./... && go build -o /dev/null ./...

# Frontend
cd frontend && pnpm lint && pnpm typecheck && pnpm test && pnpm build

# Infra (just compile)
cd infra && pnpm build
```

---

## Related Documentation

- [Tech Stack](../reference/tech-stack.md) — AWS services used
- [NFR: Security](../requirements/nfr-security.md) — Security requirements
- [Troubleshooting](troubleshooting.md) — Common issues and solutions
