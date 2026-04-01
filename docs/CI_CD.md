# CI/CD Pipeline

Multi-ecosystem deployment pipeline documentation for DS apps.

---

## Overview

The CI/CD pipeline supports deploying a single app to multiple domain ecosystems (e.g., digistratum.com, leapkick.com) from one codebase. It:

1. **Parses** app's `ecosystems.yaml` to determine enabled ecosystems
2. **Fetches** the central ecosystem registry from S3 for domain/certificate info
3. **Deploys** data stacks per ecosystem (data isolation)
4. **Deploys** a single app stack serving all ecosystem domains

### Deployment Flow

```
Push to main/develop
    │
    ▼
Parse ecosystems.yaml
    │
    ▼
Fetch ecosystem registry (S3)
    │
    ▼
Merge app config with registry
    │
    ▼
Deploy DataStacks (parallel, per ecosystem-env)
    │
    ▼
Deploy AppStack (single CF distribution, all domains)
    │
    ▼
Post-deploy validation
```

---

## Required IAM Permissions

The GitHub Actions OIDC role needs these permissions for multi-ecosystem deployments:

### S3: Ecosystem Registry Access

The pipeline fetches the central ecosystem registry from S3 at deploy time. The OIDC role **must** have read access:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EcosystemRegistryRead",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject"
      ],
      "Resource": "arn:aws:s3:::ds-infra-config/ecosystems.json"
    }
  ]
}
```

**Why this is needed:**
- The registry contains domain names, ACM certificate ARNs, and Route53 zone IDs
- Apps reference ecosystems by name only; the registry provides infrastructure details
- Without this, deployments fail at the `fetch-ecosystem-registry` job

### CDK Deployment Permissions

Standard CDK deployment permissions (CloudFormation, Lambda, S3, DynamoDB, CloudFront, ACM, Route53).

See: [AWS CDK Bootstrap](https://docs.aws.amazon.com/cdk/latest/guide/bootstrapping.html)

---

## Required GitHub Configuration

### Secrets

| Secret | Description | Required |
|--------|-------------|----------|
| `AWS_ROLE_ARN` | IAM role ARN for OIDC authentication | Yes |
| `AWS_ACCOUNT_ID` | 12-digit AWS account number | Yes |
| `NPM_TOKEN` | For private npm packages (if using) | Optional |

### Environments

| Environment | Branch | Protection |
|-------------|--------|------------|
| `development` | `develop` | None |
| `production` | `main` | Required reviewers recommended |

---

## Ecosystem Configuration

Apps declare ecosystem participation in `ecosystems.yaml`:

```yaml
version: 1

app:
  name: myapp
  displayName: "My Application"

ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: myapp

  - name: leapkick
    enabled: true
    sso_app_id: myapp
```

See [ECOSYSTEMS.md](./ECOSYSTEMS.md) for full schema reference.

### Central Registry

The central ecosystem registry at `s3://ds-infra-config/ecosystems.json` contains:

```json
{
  "ecosystems": {
    "digistratum": {
      "domains": {
        "prod": "digistratum.com",
        "dev": "dev.digistratum.com"
      },
      "acm_cert_arn": "arn:aws:acm:...",
      "route53_zone_id": "Z..."
    },
    "leapkick": {
      "domains": {
        "prod": "leapkick.com",
        "dev": "dev.leapkick.com"
      },
      "acm_cert_arn": "arn:aws:acm:...",
      "route53_zone_id": "Z..."
    }
  }
}
```

Apps never need to know infrastructure details—they just declare participation.

---

## Stack Architecture

With multi-ecosystem support, the pipeline creates:

```
├── {app}-data-{env}-{ecosystem}   # Per ecosystem-env data isolation
│   └── DynamoDB, S3, etc.
│
└── {app}-app-{env}                # Single CF distribution
    └── CloudFront, Lambda, API Gateway
        └── All ecosystem domains as aliases
```

**Example for `marketplace` with 2 ecosystems:**

```
├── marketplace-data-dev-digistratum
├── marketplace-data-dev-leapkick
├── marketplace-app-dev
│
├── marketplace-data-prod-digistratum
├── marketplace-data-prod-leapkick
└── marketplace-app-prod
```

---

## Troubleshooting

### Permission Denied on Registry Fetch

**Symptom:**
```
::error::S3 Access Denied fetching ecosystem registry
::error::The OIDC role needs s3:GetObject permission on arn:aws:s3:::ds-infra-config/ecosystems.json
```

Or in AWS CLI output:
```
An error occurred (AccessDenied) when calling the GetObject operation: Access Denied
```

**Cause:** The OIDC role lacks `s3:GetObject` permission on the registry bucket.

**Solution:**

1. Verify the IAM role has the required S3 policy (see [Required IAM Permissions](#s3-ecosystem-registry-access))
2. Check the role ARN matches `AWS_ROLE_ARN` secret
3. Ensure the trust policy allows your repo:
   ```json
   {
     "Effect": "Allow",
     "Principal": {
       "Federated": "arn:aws:iam::ACCOUNT:oidc-provider/token.actions.githubusercontent.com"
     },
     "Action": "sts:AssumeRoleWithWebIdentity",
     "Condition": {
       "StringEquals": {
         "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
       },
       "StringLike": {
         "token.actions.githubusercontent.com:sub": "repo:YourOrg/your-repo:*"
       }
     }
   }
   ```

### Ecosystem Not Found in Registry

**Symptom:**
```
::error::Ecosystems not found in registry: newdomain
```

**Cause:** The app's `ecosystems.yaml` references an ecosystem not in the central registry.

**Solution:**

1. Check spelling matches exactly (e.g., `digistratum` not `DigiStratum`)
2. Verify the ecosystem exists in `s3://ds-infra-config/ecosystems.json`
3. If adding a new ecosystem, update the central registry first

### OIDC Authentication Failed

**Symptom:**
```
Error: Could not assume role with OIDC
```

**Solution:**

1. Verify `AWS_ROLE_ARN` secret is set correctly
2. Check IAM role trust policy includes your repo and branch
3. Ensure OIDC provider is configured in AWS account

### Registry Unavailable (Degraded Mode)

**Symptom:**
```
::warning::Registry unavailable - proceeding without validation
```

**Cause:** S3 fetch failed but pipeline continues with fallback.

**Impact:** Ecosystems won't be validated against registry. Deploy may fail later if ecosystem details are wrong.

**Solution:**

1. Check AWS service health
2. Verify S3 bucket exists and is accessible
3. Retry the workflow

---

## Environment Detection

The pipeline automatically determines the target environment:

| Trigger | Branch | Environment |
|---------|--------|-------------|
| `push` | `main` | `prod` |
| `push` | `develop` | `dev` |
| `workflow_dispatch` | Any | Specified in input |
| `pull_request` | Any | None (build only) |

### Manual Deployment

Use workflow_dispatch for manual deployments:

```bash
gh workflow run ci-cd.yml -f environment=prod
```

---

## E2E Testing Gate

After successful dev deployment, the `e2e-dev.yml` workflow runs end-to-end tests. This acts as a **promotion gate** — production deploys are blocked until E2E tests pass.

### E2E Workflow Flow

```
Dev Deploy (ci-cd.yml)
    │
    ▼ (workflow_run trigger)
E2E Tests (e2e-dev.yml)
    │
    ├─ API tests (tests/e2e/api/)
    ├─ UI tests (tests/e2e/ui/)
    └─ Security tests (tests/e2e/security/)
    │
    ▼
Commit status: e2e/dev
    │
    ├─ ✅ success → Ready for prod promotion
    └─ ❌ failure → Blocked (requires fix + re-run)
```

### E2E Secrets

| Secret | Description | Required |
|--------|-------------|----------|
| `E2E_TEST_USER_EMAIL` | Test user email for authenticated tests | Yes* |
| `E2E_TEST_USER_PASSWORD` | Test user password | Yes* |
| `E2E_SESSION_TOKEN` | Pre-authenticated session token | Alternative |

*Either email/password pair OR session token is required.

### Branch Protection Configuration

To enforce E2E tests before prod deployment:

1. Go to **Settings → Branches → main**
2. Enable **Require status checks to pass**
3. Add required check: `e2e/dev`
4. Enable **Require branches to be up to date**

With this configuration:
- PRs to `main` must have passing E2E tests
- Direct pushes to `main` still run E2E via workflow_run

### Manual E2E Run

Trigger E2E tests manually for debugging:

```bash
# With default dev URL
gh workflow run e2e-dev.yml

# With custom URL
gh workflow run e2e-dev.yml -f app_url=https://custom.dev.digistratum.com
```

### Test Artifacts

On failure, the workflow uploads:
- Screenshots (`test-results/**/screenshot*.png`)
- Traces (`test-results/**/trace*.zip`)
- Videos (`test-results/**/video*.webm`)

Download via:
```bash
gh run download <run-id> -n e2e-failure-artifacts-<run-number>
```

### Adding New E2E Tests

Tests follow the structure in `tests/e2e/`:

| Directory | Purpose |
|-----------|---------|
| `api/` | REST API endpoint tests |
| `ui/` | Playwright browser tests |
| `security/` | Auth/authz edge cases |
| `fixtures/` | Shared test helpers |
| `smoke.test.ts` | Production smoke tests (`@smoke` tagged) |

See [tests/e2e/README.md](../tests/e2e/README.md) for conventions.

---

## Production Smoke Tests & Rollback

After successful production deployment, `smoke-prod.yml` runs critical-path smoke tests. On failure, it automatically rolls back to the previous Lambda version.

### Smoke Test Flow

```
Prod Deploy (ci-cd.yml main branch)
    │
    ▼ (workflow_run trigger)
Capture Lambda Versions
    │
    ▼
Smoke Tests (smoke-prod.yml)
    │
    ├─ Only @smoke tagged tests
    ├─ Health check
    ├─ Auth flow
    └─ Core pages load
    │
    ▼
    ├─ ✅ success → Mark deployment complete
    └─ ❌ failure → Trigger rollback
                     │
                     ▼
              Lambda alias → previous version
                     │
                     ▼
              CloudFront invalidation (API paths)
                     │
                     ▼
              Slack alert (if configured)
```

### Smoke Test Tags

Tag tests with `@smoke` for inclusion in prod smoke suite:

```typescript
/**
 * @smoke
 * @covers NFR-100 Service availability
 */
it('@smoke health endpoint returns 200', async () => {
  const response = await client.get('/api/health');
  expect(response.status).toBe(200);
});
```

Run smoke tests locally:
```bash
npx playwright test tests/e2e/ --grep "@smoke"
```

### Rollback Mechanism

On smoke test failure:

1. **Lambda Rollback**: Updates the `live` alias to point to previous version
2. **CloudFront Invalidation**: Clears `/api/*` paths (optional, if distribution exists)
3. **Alert**: Posts to Slack webhook (if `SLACK_WEBHOOK_URL` configured)
4. **Status**: Updates commit status to `failure`

**Prerequisites for rollback:**
- Lambda must use aliases (CDK default: `live` alias)
- At least one previous version must exist
- OIDC role must have `lambda:UpdateAlias` permission

### Lambda Alias Setup

The rollback relies on Lambda alias versioning. CDK templates should configure:

```typescript
const fn = new lambda.Function(this, 'ApiFunction', { ... });

// Create version for each deployment
const version = fn.currentVersion;

// Create or update 'live' alias
new lambda.Alias(this, 'LiveAlias', {
  aliasName: 'live',
  version: version,
});

// API Gateway points to alias, not $LATEST
const integration = new apigateway.LambdaIntegration(
  lambda.Function.fromFunctionArn(this, 'AliasRef', `${fn.functionArn}:live`)
);
```

### Required Permissions

Add to OIDC role policy for rollback:

```json
{
  "Effect": "Allow",
  "Action": [
    "lambda:GetAlias",
    "lambda:UpdateAlias",
    "lambda:ListVersionsByFunction"
  ],
  "Resource": "arn:aws:lambda:*:*:function:*"
}
```

### Manual Smoke Test Run

```bash
# Run against prod URL
gh workflow run smoke-prod.yml

# Custom URL with skip-rollback (dry run)
gh workflow run smoke-prod.yml -f app_url=https://myapp.digistratum.com -f skip_rollback=true
```

### Monitoring Rollbacks

Rollback events appear in:
- GitHub Actions run summary
- Slack channel (if webhook configured)
- Commit status: `smoke/prod`

Check rollback history:
```bash
# List recent Lambda alias updates
aws lambda get-alias --function-name myapp-api-prod --name live

# List all versions
aws lambda list-versions-by-function --function-name myapp-api-prod
```

---

## Related Documentation

- [ECOSYSTEMS.md](./ECOSYSTEMS.md) — Schema reference for ecosystems.yaml
- [Deployment Runbook](./runbooks/deployment.md) — General deployment guide
- [Troubleshooting](./runbooks/troubleshooting.md) — Common issues
