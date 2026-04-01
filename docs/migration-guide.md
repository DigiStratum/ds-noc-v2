# Migration Guide: Multi-Ecosystem Pattern

Step-by-step guide for migrating existing DS apps to the multi-ecosystem deployment pattern.

## Overview

This guide walks through migrating existing apps (like DSAccount, DSKanban) to support multiple ecosystems (digistratum.com, leapkick.com) from a single codebase.

**Migration time estimates:**
| Phase | Duration | Risk Level |
|-------|----------|------------|
| Phase 1: Configuration | 1-2 hours | Low |
| Phase 2: Stack Deployment | 2-4 hours | Medium |
| Phase 3: Data Sync | 4-24 hours* | Medium |
| Phase 4: Cutover | 1-2 hours | High |

*Depends on data volume

## Prerequisites Checklist

Complete these before starting migration:

### AWS Access
- [ ] AWS CLI configured with appropriate credentials
- [ ] Access to deploy CloudFormation stacks
- [ ] Access to DynamoDB (read existing, create new tables)
- [ ] Access to S3 (`ds-infra-config` bucket for registry)
- [ ] Route53 permissions for DNS validation

### Local Environment
- [ ] Node.js 18+ installed
- [ ] CDK CLI installed (`npm install -g aws-cdk`)
- [ ] Go 1.24+ installed (for backend)
- [ ] pnpm installed (for workspace)
- [ ] Git with write access to the app repo

### Ecosystem Registry
- [ ] App registered in central registry (`s3://ds-infra-config/ecosystems.yaml`)
- [ ] ACM certificates available for target ecosystems
- [ ] SSO app IDs registered with DSAccount (all target ecosystems)

### Pre-Migration Backup
- [ ] DynamoDB point-in-time recovery enabled on existing tables
- [ ] Recent DynamoDB export to S3 completed
- [ ] Current CloudFormation template exported

```bash
# Export current stack
aws cloudformation get-template \
  --stack-name dsaccount-prod \
  --query 'TemplateBody' > backup/dsaccount-prod-template.json

# Verify PITR is enabled
aws dynamodb describe-continuous-backups \
  --table-name dsaccount-prod
```

---

## Phase 1: Add ecosystems.yaml

**Goal:** Add explicit ecosystem configuration to your app without changing deployment.

**Duration:** 1-2 hours  
**Risk:** Low — existing deployments unchanged

### Step 1.1: Create ecosystems.yaml

In your app root, create `ecosystems.yaml`:

```yaml
# ecosystems.yaml
version: 1

app:
  name: dsaccount
  displayName: "DS Account"

ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: dsaccount
```

**For DSKanban:**
```yaml
version: 1

app:
  name: dskanban
  displayName: "DS Projects"

ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: projects
```

### Step 1.2: Verify Stack Names Unchanged

Single-ecosystem mode preserves existing stack names:

```bash
cd infra

# Compare current deployed stack
aws cloudformation list-stacks \
  --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE \
  | jq -r '.StackSummaries[] | select(.StackName | startswith("dsaccount")) | .StackName'

# Verify CDK produces same name
cdk synth --context env=prod 2>&1 | grep "Stack:"
```

**Expected:** Stack name should be identical (e.g., `dsaccount-prod`).

⚠️ **STOP if stack names differ.** Do not proceed — investigate why CDK is producing different names.

### Step 1.3: Commit Configuration

```bash
git add ecosystems.yaml
git commit -m "Add ecosystems.yaml for multi-ecosystem support (Phase 1)"
git push
```

### Phase 1 Rollback

If issues arise:
```bash
git rm ecosystems.yaml
git commit -m "Rollback: Remove ecosystems.yaml"
git push
```

No AWS changes needed — configuration only.

---

## Phase 2: Deploy New Stack Structure

**Goal:** Deploy multi-ecosystem stacks alongside existing stack.

**Duration:** 2-4 hours  
**Risk:** Medium — new stacks created, existing stack preserved

### Step 2.1: Enable Multi-Ecosystem

Update `ecosystems.yaml` to add second ecosystem:

```yaml
version: 1

app:
  name: dsaccount
  displayName: "DS Account"

ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: dsaccount
  
  - name: leapkick
    enabled: true
    sso_app_id: dsaccount  # Same app ID or different if registered separately
```

### Step 2.2: Preview Stack Changes

```bash
cd infra

# List new stacks that will be created
cdk ls --context env=prod

# Expected output for multi-ecosystem:
#   dsaccount-app-prod
#   dsaccount-data-prod-digistratum
#   dsaccount-data-prod-leapkick
```

**Stacks created:**
| Stack | Purpose |
|-------|---------|
| `{app}-app-{env}` | CloudFront distribution (all ecosystems), Lambda, API Gateway |
| `{app}-data-{env}-{eco}` | DynamoDB tables, S3 buckets (per ecosystem) |

### Step 2.3: Deploy Data Stacks First

Deploy data stacks before the app stack to ensure tables exist:

```bash
# Deploy ecosystem-specific data stacks
cdk deploy dsaccount-data-prod-digistratum --context env=prod
cdk deploy dsaccount-data-prod-leapkick --context env=prod
```

**Verify tables created:**
```bash
aws dynamodb list-tables | jq -r '.TableNames[]' | grep dsaccount
# Expected:
#   dsaccount-prod (existing)
#   dsaccount-prod-digistratum (new)
#   dsaccount-prod-leapkick (new)
```

### Step 2.4: Deploy App Stack

⚠️ **Important:** The app stack will create a NEW CloudFront distribution with different domain routing. The old stack (`dsaccount-prod`) remains unchanged but will NOT serve traffic once the new distribution is active.

```bash
# Deploy the unified app stack
cdk deploy dsaccount-app-prod --context env=prod
```

### Step 2.5: Verify Deployment

```bash
# Check all stacks healthy
aws cloudformation describe-stacks \
  --query 'Stacks[?starts_with(StackName, `dsaccount`)].{Name:StackName,Status:StackStatus}' \
  --output table

# Verify CloudFront distribution
aws cloudfront list-distributions \
  --query 'DistributionList.Items[?Comment==`dsaccount-app-prod`]'
```

### Phase 2 Rollback

If deployment fails or issues arise:

```bash
# Remove new stacks (keeps old stack intact)
cdk destroy dsaccount-app-prod --context env=prod --force
cdk destroy dsaccount-data-prod-leapkick --context env=prod --force
cdk destroy dsaccount-data-prod-digistratum --context env=prod --force

# Revert ecosystems.yaml to single ecosystem
git checkout HEAD~1 -- ecosystems.yaml
git commit -m "Rollback: Revert to single ecosystem"
git push
```

Old stack continues serving traffic unchanged.

---

## Phase 3: Data Sync (DynamoDB Streams)

**Goal:** Sync data from old tables to new ecosystem-specific tables.

**Duration:** 4-24 hours (depends on data volume)  
**Risk:** Medium — data replication, must verify consistency

### Step 3.1: Enable DynamoDB Streams on Old Table

```bash
aws dynamodb update-table \
  --table-name dsaccount-prod \
  --stream-specification StreamEnabled=true,StreamViewType=NEW_AND_OLD_IMAGES
```

### Step 3.2: Deploy Migration Lambda

The migration helper Lambda reads from the stream and writes to the new tables:

```bash
# From ds-app-template tools
cd tools
./deploy-migration-lambda.sh dsaccount prod

# Or manually deploy the Lambda
aws lambda create-function \
  --function-name dsaccount-migration-stream-handler \
  --runtime provided.al2023 \
  --handler bootstrap \
  --role arn:aws:iam::${ACCOUNT}:role/dsaccount-migration-role \
  --zip-file fileb://migration-lambda.zip \
  --environment Variables='{
    "SOURCE_TABLE":"dsaccount-prod",
    "TARGET_TABLES":"dsaccount-prod-digistratum,dsaccount-prod-leapkick",
    "ECOSYSTEM_ROUTING":"tenant_domain"
  }'
```

### Step 3.3: Create Event Source Mapping

```bash
STREAM_ARN=$(aws dynamodb describe-table \
  --table-name dsaccount-prod \
  --query 'Table.LatestStreamArn' --output text)

aws lambda create-event-source-mapping \
  --function-name dsaccount-migration-stream-handler \
  --event-source-arn "$STREAM_ARN" \
  --starting-position TRIM_HORIZON \
  --batch-size 100
```

### Step 3.4: Backfill Historical Data

Stream only captures new changes. Run backfill for existing data:

```bash
# Using the migration helper script
cd tools
./backfill-ecosystem-data.sh dsaccount prod

# This scans the source table and writes to ecosystem-specific tables
# based on the ecosystem_routing strategy (e.g., tenant_domain)
```

**Backfill strategies:**
| Strategy | Description |
|----------|-------------|
| `tenant_domain` | Route based on tenant's registered domain |
| `partition_key_prefix` | Route based on PK prefix (e.g., `ds#` vs `lk#`) |
| `duplicate_all` | Copy all data to all ecosystems (for shared reference data) |

### Step 3.5: Verify Data Consistency

```bash
# Run validation script
cd tools
./validate-migration.sh dsaccount prod

# Manual verification
aws dynamodb scan --table-name dsaccount-prod --select COUNT
aws dynamodb scan --table-name dsaccount-prod-digistratum --select COUNT
aws dynamodb scan --table-name dsaccount-prod-leapkick --select COUNT
```

**Validation checkpoints:**
- [ ] Record counts match (accounting for routing)
- [ ] Sample records verified in each ecosystem table
- [ ] Stream lag is zero (all changes replicated)
- [ ] Application reads work from new tables

### Phase 3 Rollback

If data sync fails:

```bash
# Remove event source mapping
aws lambda delete-event-source-mapping \
  --uuid $(aws lambda list-event-source-mappings \
    --function-name dsaccount-migration-stream-handler \
    --query 'EventSourceMappings[0].UUID' --output text)

# Delete migration Lambda
aws lambda delete-function --function-name dsaccount-migration-stream-handler

# Disable streams on source table (optional, streams auto-expire)
aws dynamodb update-table \
  --table-name dsaccount-prod \
  --stream-specification StreamEnabled=false
```

Data in new tables can be deleted or retained for retry.

---

## Phase 4: Cutover and Cleanup

**Goal:** Switch traffic to new stacks and remove old stack.

**Duration:** 1-2 hours  
**Risk:** High — production traffic switch

### Step 4.1: Pre-Cutover Checklist

- [ ] Data validation passed (Phase 3.5)
- [ ] New CloudFront distribution healthy
- [ ] New Lambda functions tested
- [ ] DNS TTLs lowered (if applicable)
- [ ] Maintenance window scheduled
- [ ] Rollback plan reviewed

### Step 4.2: Cutover Steps

**Option A: DNS Cutover (Zero-Downtime)**

If using Route53 with weighted routing:

```bash
# Gradually shift traffic
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567 \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "dsaccount.digistratum.com",
        "Type": "A",
        "SetIdentifier": "new-stack",
        "Weight": 100,
        "AliasTarget": {
          "HostedZoneId": "Z2FDTNDATAQYW2",
          "DNSName": "d111111abcdef.cloudfront.net",
          "EvaluateTargetHealth": true
        }
      }
    }]
  }'
```

**Option B: Stack Replacement**

Update the app stack to use new resources directly:

```bash
cd infra
cdk deploy dsaccount-app-prod --context env=prod
```

### Step 4.3: Verify Cutover

```bash
# Test endpoints
curl -I https://dsaccount.digistratum.com/api/health
curl -I https://dsaccount.leapkick.com/api/health

# Verify correct CloudFront distribution
curl -s -I https://dsaccount.digistratum.com | grep -i x-amz-cf-pop

# Check Lambda logs
aws logs tail /aws/lambda/dsaccount-app-prod --since 5m
```

### Step 4.4: Cleanup Old Stack

⚠️ **Wait 24-48 hours** after successful cutover before cleanup.

```bash
# Final validation
./validate-migration.sh dsaccount prod --final

# Disable stream handler
aws lambda delete-event-source-mapping \
  --uuid $(aws lambda list-event-source-mappings \
    --function-name dsaccount-migration-stream-handler \
    --query 'EventSourceMappings[0].UUID' --output text)

# Delete migration Lambda
aws lambda delete-function --function-name dsaccount-migration-stream-handler

# Export old table (final backup)
aws dynamodb export-table-to-point-in-time \
  --table-arn arn:aws:dynamodb:us-east-1:${ACCOUNT}:table/dsaccount-prod \
  --s3-bucket ds-backups \
  --s3-prefix final-migration-backup/dsaccount-prod

# Delete old stack
cdk destroy dsaccount-prod --context env=prod --force
```

### Phase 4 Rollback

If cutover fails and you need to revert:

**Within first hour:**
```bash
# Revert DNS (if using DNS cutover)
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567 \
  --change-batch '...'  # Restore old record

# Or redeploy old stack
cdk deploy dsaccount-prod --context env=prod
```

**After cleanup (old stack deleted):**
```bash
# Restore from CloudFormation export
aws cloudformation create-stack \
  --stack-name dsaccount-prod \
  --template-body file://backup/dsaccount-prod-template.json

# Restore DynamoDB data
aws dynamodb restore-table-from-backup \
  --target-table-name dsaccount-prod \
  --backup-arn arn:aws:dynamodb:us-east-1:${ACCOUNT}:table/dsaccount-prod/backup/...
```

---

## App-Specific Notes

### DSAccount Migration

DSAccount is the SSO provider — migration requires extra care:

1. **Coordinate with dependent apps** — All apps authenticate via DSAccount
2. **Session continuity** — Users should not need to re-login
3. **Cookie domain** — `.digistratum.com` cookie works for both ecosystems
4. **LeapKick SSO** — Requires separate session cookie (`.leapkick.com`)

**Recommended approach:**
- Deploy LeapKick ecosystem as new tenant, not migration
- Existing digistratum users unaffected
- New leapkick.com users register fresh

### DSKanban Migration

DSKanban stores project/issue data per tenant:

1. **Tenant routing** — Each tenant belongs to one ecosystem
2. **No cross-ecosystem data** — Projects in digistratum stay there
3. **API keys** — May need ecosystem-specific keys

**Data routing strategy:** `tenant_domain` — route based on tenant's domain.

---

## Tooling Reference

These scripts support migration (from `ds-app-template/tools/`):

| Script | Purpose | Phase |
|--------|---------|-------|
| `generate-ecosystems-yaml.sh` | Create initial config from existing app | 1 |
| `deploy-migration-lambda.sh` | Deploy DynamoDB Streams handler | 3 |
| `backfill-ecosystem-data.sh` | Copy historical data to new tables | 3 |
| `validate-migration.sh` | Compare data between old/new tables | 3, 4 |
| `cleanup-old-stack.sh` | Remove old resources after cutover | 4 |

---

## Troubleshooting

### "Ecosystem not found in registry"

Check that your ecosystem name matches the central registry exactly.

```bash
aws s3 cp s3://ds-infra-config/ecosystems.yaml - | grep "name:"
```

### "Stack already exists with different name"

Multi-ecosystem mode changes stack naming. Follow Phase 2 carefully — deploy new stacks alongside old, don't update in place.

### "DynamoDB Streams lag increasing"

The migration Lambda may be throttled:

```bash
# Check Lambda errors
aws logs filter-log-events \
  --log-group-name /aws/lambda/dsaccount-migration-stream-handler \
  --filter-pattern ERROR

# Increase Lambda concurrency
aws lambda put-function-concurrency \
  --function-name dsaccount-migration-stream-handler \
  --reserved-concurrent-executions 10
```

### "Data mismatch between tables"

Run the validation script with verbose output:

```bash
./validate-migration.sh dsaccount prod --verbose --sample 100
```

---

## Related Documentation

- [ECOSYSTEMS.md](./ECOSYSTEMS.md) — ecosystems.yaml schema reference
- [MIGRATION-MULTI-ECOSYSTEM.md](./MIGRATION-MULTI-ECOSYSTEM.md) — Quick reference for multi-ecosystem CDK changes
- [CI_CD.md](./CI_CD.md) — Deployment pipeline configuration
- [PLAN #1749](https://projects.digistratum.com/issues/1749) — Original architecture decisions
