# Migration Guide: Multi-Ecosystem Support

This guide covers upgrading existing single-ecosystem apps to the new multi-ecosystem CDK infrastructure.

## Overview

The multi-ecosystem update adds:
- `ecosystems.yaml` configuration file
- Central registry for ecosystem metadata
- DataStack separation for multi-ecosystem data isolation

**Key guarantee:** Single-ecosystem apps continue working unchanged.

## Migration Scenarios

### Scenario 1: Keep Single-Ecosystem (No Changes Required)

If your app only needs to serve one ecosystem (e.g., digistratum.com):

**Option A: Do nothing**
- Apps without `ecosystems.yaml` continue working in legacy mode
- Stack naming unchanged: `{app}-{env}`
- No registry dependency

**Option B: Add explicit ecosystems.yaml** (recommended for clarity)

Create `ecosystems.yaml` in your app root:

```yaml
version: 1

app:
  name: myapp
  displayName: "My Application"

ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: myapp
```

Benefits:
- Explicit configuration vs implicit defaults
- Enables future multi-ecosystem expansion
- **Stack naming remains identical:** `{app}-{env}`

### Scenario 2: Upgrade to Multi-Ecosystem

To serve multiple ecosystems (e.g., digistratum.com + leapkick.com):

1. **Create ecosystems.yaml:**

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

2. **Deploy with new stack structure:**

⚠️ **Breaking change:** Multi-ecosystem uses different stack names:
- Before: `myapp-prod`
- After: `myapp-app-prod`, `myapp-data-prod-digistratum`, `myapp-data-prod-leapkick`

**Migration steps:**

```bash
# 1. Export existing data (if needed)
aws dynamodb export-table-to-point-in-time \
  --table-arn arn:aws:dynamodb:region:account:table/myapp-prod \
  --s3-bucket my-backup-bucket

# 2. Deploy new stacks
cd infra
cdk deploy --all

# 3. Migrate data to ecosystem-specific tables
# (app-specific migration script)

# 4. Remove old stack (after verification)
cdk destroy myapp-prod
```

## Stack Naming Reference

| Configuration | Stack Names |
|--------------|-------------|
| No ecosystems.yaml (legacy) | `{app}-{env}` |
| ecosystems.yaml with 1 ecosystem | `{app}-{env}` |
| ecosystems.yaml with 2+ ecosystems | `{app}-app-{env}`, `{app}-data-{env}-{eco}` |

## Data Isolation

### Single-Ecosystem
- One DynamoDB table: `{app}-{env}`
- One S3 bucket: `{app}-{env}-frontend-{account}`
- Created by AppStack

### Multi-Ecosystem
- DynamoDB tables per ecosystem: `{app}-{env}-{ecosystem}`
- S3 buckets per ecosystem: `{app}-{env}-{ecosystem}-assets-{account}`
- Created by DataStack (separate stack per ecosystem)

## Environment Variables

### Single-Ecosystem (Legacy)
```
DYNAMODB_TABLE=myapp-prod
APP_URL=https://myapp.digistratum.com
DSACCOUNT_SSO_URL=https://account.digistratum.com
```

### Multi-Ecosystem
```
ECOSYSTEM_CONFIG={"digistratum":{"domain":"...","tableName":"..."},...}
```

Lambda receives `ECOSYSTEM_CONFIG` as JSON containing per-ecosystem settings. The app determines the current ecosystem from the request's Host header.

## Registry Dependency

Multi-ecosystem mode requires the central registry:
- S3: `s3://ds-infra-config/ecosystems.yaml`
- Contains: certificates, zone IDs, SSO URLs per ecosystem

Single-ecosystem mode (including explicit single-ecosystem config) fetches from the registry only to resolve the ecosystem's infrastructure details.

## Testing Your Migration

### Verify CDK Synth

```bash
cd infra

# Single-ecosystem should produce one stack
cdk synth --context env=dev
# Output: myapp-dev.template.json

# Multi-ecosystem should produce multiple stacks
cdk ls --context env=dev
# Output:
#   myapp-app-dev
#   myapp-data-dev-digistratum
#   myapp-data-dev-leapkick
```

### Verify Stack Names Won't Change (Single-Ecosystem)

If you're adding ecosystems.yaml to an existing deployed app:

```bash
# Get current stack name from CloudFormation
aws cloudformation list-stacks --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE \
  | jq -r '.StackSummaries[] | select(.StackName | contains("myapp")) | .StackName'

# Run CDK synth and verify stack name matches
cd infra
cdk synth --context env=prod 2>&1 | grep -E "^Stack:"
```

The stack name should be identical. If not, do NOT deploy—it will replace the stack.

## Rollback

If migration fails:

1. **Restore ecosystems.yaml** to single-ecosystem (or remove it)
2. **CDK deploy** reverts to single-stack structure
3. **Restore data** from export if needed

## Common Issues

### "Ecosystem not found in registry"
The ecosystem name in your ecosystems.yaml must match the registry exactly.
Check available ecosystems: `digistratum`, `leapkick`

### "Stack already exists with different name"
This happens when adding multi-ecosystem to an existing app.
Multi-ecosystem requires stack rename—see Scenario 2 migration steps.

### AWS credentials not configured
Registry fetch requires AWS credentials with S3 read access to `ds-infra-config` bucket.

## Support

For issues with multi-ecosystem migration:
1. Check this guide first
2. See [ECOSYSTEMS.md](./ECOSYSTEMS.md) for configuration details
3. File an issue in ds-app-template repo
