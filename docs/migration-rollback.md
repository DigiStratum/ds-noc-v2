# Migration Rollback Procedures

Detailed rollback procedures for each phase of multi-ecosystem migration. Use this guide when migration encounters issues requiring reversal.

**Related:** [Migration Guide](./migration-guide.md) | [ECOSYSTEMS.md](./ECOSYSTEMS.md)

---

## Decision Framework: Rollback vs Push Forward

Before rolling back, evaluate:

| Factor | Push Forward | Roll Back |
|--------|--------------|-----------|
| **Data integrity** | Data verified consistent | Data corruption detected |
| **Partial failure** | Some stacks succeeded, fixable | Cascading failures |
| **Time pressure** | Off-peak, runway available | Production impact, users affected |
| **Root cause** | Known and addressable | Unknown or complex |
| **Rollback risk** | Higher than proceeding | Lower than proceeding |

**General rule:** If you can identify and fix the issue in less time than rollback + retry, push forward. If unsure, roll back — migrations are designed to be repeatable.

---

## Time Estimates Summary

| Phase | Rollback Duration | Complexity | Data Risk |
|-------|-------------------|------------|-----------|
| Phase 1 | 5 minutes | Trivial | None |
| Phase 2 | 15-30 minutes | Low | None |
| Phase 3 | 30-60 minutes | Medium | Low |
| Phase 4 (early) | 30-60 minutes | Medium | Medium |
| Phase 4 (post-cleanup) | 2-4 hours | High | High |

---

## Phase 1 Rollback: Configuration Only

**Scenario:** ecosystems.yaml added but causing CDK issues or validation failures.

**Risk level:** None — no infrastructure changes made.

**Duration:** ~5 minutes

### When to Rollback Phase 1

- CDK synth produces different stack names than expected
- Validation errors in ecosystems.yaml schema
- Decision to postpone migration

### Rollback Steps

```bash
cd ~/repos/digistratum/<app>

# Remove configuration file
git rm ecosystems.yaml
git commit -m "Rollback: Remove ecosystems.yaml (migration postponed)"
git push

# Verify CDK unchanged
cd infra
cdk synth --context env=prod 2>&1 | grep "Stack:"
# Should match pre-migration stack names exactly
```

### Verification

```bash
# Confirm no AWS changes needed
aws cloudformation describe-stacks \
  --stack-name <app>-prod \
  --query 'Stacks[0].StackStatus'
# Should show CREATE_COMPLETE or UPDATE_COMPLETE
```

### Recovery: None needed

Phase 1 is config-only. Re-attempt by recreating ecosystems.yaml.

---

## Phase 2 Rollback: Stack Deployment

**Scenario:** New stacks deployed alongside existing, but new stacks failing or misconfigured.

**Risk level:** Low — old stacks untouched, still serving traffic.

**Duration:** 15-30 minutes

### When to Rollback Phase 2

- Data stack deployment failed (DynamoDB table creation issues)
- App stack deployment failed (Lambda/API Gateway issues)
- CloudFront distribution not propagating correctly
- Certificate validation failures
- Need to abort before Phase 3

### Rollback Steps

```bash
cd ~/repos/digistratum/<app>/infra

# Destroy new stacks in reverse order (app first, then data)
cdk destroy <app>-app-prod --context env=prod --force

# Destroy ecosystem-specific data stacks
cdk destroy <app>-data-prod-leapkick --context env=prod --force
cdk destroy <app>-data-prod-digistratum --context env=prod --force

# If CDK destroy hangs or fails, use CloudFormation directly
aws cloudformation delete-stack --stack-name <app>-app-prod
aws cloudformation wait stack-delete-complete --stack-name <app>-app-prod
aws cloudformation delete-stack --stack-name <app>-data-prod-leapkick
aws cloudformation delete-stack --stack-name <app>-data-prod-digistratum
```

### Revert Configuration

```bash
# Return to single-ecosystem config (or remove entirely)
cd ~/repos/digistratum/<app>

# Option A: Revert to single ecosystem
cat > ecosystems.yaml << 'EOF'
version: 1

app:
  name: <app>
  displayName: "<Display Name>"

ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: <app>
EOF

git add ecosystems.yaml
git commit -m "Rollback: Revert to single ecosystem configuration"
git push

# Option B: Remove entirely
git rm ecosystems.yaml
git commit -m "Rollback: Remove ecosystems.yaml"
git push
```

### Verification

```bash
# Confirm old stack still healthy
aws cloudformation describe-stacks \
  --stack-name <app>-prod \
  --query 'Stacks[0].{Name:StackName,Status:StackStatus}'

# Confirm new stacks deleted
aws cloudformation list-stacks \
  --stack-status-filter DELETE_COMPLETE \
  --query 'StackSummaries[?starts_with(StackName, `<app>-data-prod`) || starts_with(StackName, `<app>-app-prod`)].StackName'

# Test production endpoint
curl -s https://<app>.digistratum.com/api/health
```

### Troubleshooting Phase 2 Rollback

**"Stack deletion failed: resource in use"**

Check for retained resources:
```bash
aws cloudformation describe-stack-resources \
  --stack-name <app>-data-prod-leapkick \
  --query 'StackResources[?ResourceStatus==`DELETE_FAILED`]'

# Manually delete retained resources, then retry
aws dynamodb delete-table --table-name <table-name>
aws cloudformation delete-stack --stack-name <app>-data-prod-leapkick
```

**"DynamoDB table has deletion protection"**

Disable protection first:
```bash
aws dynamodb update-table \
  --table-name <app>-prod-leapkick \
  --deletion-protection-enabled false

aws dynamodb delete-table --table-name <app>-prod-leapkick
```

---

## Phase 3 Rollback: Data Sync

**Scenario:** DynamoDB Streams replication running but data issues detected.

**Risk level:** Low to Medium — old table remains authoritative.

**Duration:** 30-60 minutes

### When to Rollback Phase 3

- Stream handler failing repeatedly
- Data routing logic incorrect (wrong ecosystem assignment)
- Backfill script errors
- Performance impact on source table
- Data validation failures

### Rollback Steps

#### Step 1: Disable Stream Handler

```bash
# Get the event source mapping UUID
MAPPING_UUID=$(aws lambda list-event-source-mappings \
  --function-name <app>-migration-stream-handler \
  --query 'EventSourceMappings[0].UUID' --output text)

# Delete the mapping (stops processing immediately)
aws lambda delete-event-source-mapping --uuid "$MAPPING_UUID"

# Verify disabled
aws lambda list-event-source-mappings \
  --function-name <app>-migration-stream-handler
# Should return empty EventSourceMappings array
```

#### Step 2: Delete Migration Lambda

```bash
aws lambda delete-function --function-name <app>-migration-stream-handler

# Delete associated log group
aws logs delete-log-group \
  --log-group-name /aws/lambda/<app>-migration-stream-handler
```

#### Step 3: Disable Streams on Source Table (Optional)

Streams auto-expire after 24 hours if not consumed. Explicit disable optional:

```bash
aws dynamodb update-table \
  --table-name <app>-prod \
  --stream-specification StreamEnabled=false

# Verify
aws dynamodb describe-table \
  --table-name <app>-prod \
  --query 'Table.StreamSpecification'
# Should show null or StreamEnabled: false
```

#### Step 4: Handle Data in New Tables

**Option A: Keep data for retry**

Leave new tables intact. On retry, backfill will overwrite:
```bash
# Just verify tables exist
aws dynamodb describe-table --table-name <app>-prod-digistratum
aws dynamodb describe-table --table-name <app>-prod-leapkick
```

**Option B: Clear new tables**

If data is corrupt or routing was wrong:
```bash
# Delete all items (for small tables)
cd ~/repos/digistratum/ds-app-template/tools
./clear-table.sh <app>-prod-digistratum
./clear-table.sh <app>-prod-leapkick

# Or delete and recreate tables
aws dynamodb delete-table --table-name <app>-prod-digistratum
aws dynamodb delete-table --table-name <app>-prod-leapkick

# Redeploy data stacks
cd ~/repos/digistratum/<app>/infra
cdk deploy <app>-data-prod-digistratum --context env=prod
cdk deploy <app>-data-prod-leapkick --context env=prod
```

### Verification

```bash
# Confirm no stream processing
aws dynamodb describe-table \
  --table-name <app>-prod \
  --query 'Table.{StreamEnabled:StreamSpecification.StreamEnabled,StreamArn:LatestStreamArn}'

# Confirm Lambda deleted
aws lambda get-function --function-name <app>-migration-stream-handler 2>&1
# Should show "Function not found"

# Confirm source table healthy
aws dynamodb scan --table-name <app>-prod --select COUNT
```

### Recovery

Re-attempt Phase 3 by:
1. Fixing the root cause (routing logic, handler code, etc.)
2. Re-enabling streams
3. Redeploying migration Lambda
4. Re-running backfill

---

## Phase 4 Rollback: Post-Cutover

**Scenario:** Traffic switched to new stacks but issues detected.

**Risk level:** Medium to High — depends on how much new data has been written.

**Duration:** 30 minutes to 4 hours, depending on data divergence.

### Decision Point: How Far Along?

| Situation | Rollback Complexity |
|-----------|---------------------|
| Cutover < 1 hour, minimal writes | Simple DNS switch |
| Cutover > 1 hour, writes occurring | Data reconciliation needed |
| Old stack deleted | Full restore required |

---

### Phase 4A: Early Rollback (Old Stack Intact)

**Scenario:** Cutover failed or issues detected within ~1 hour, old stack still running.

**Duration:** 30-60 minutes

#### DNS Rollback (If Using Weighted Routing)

```bash
# Switch all traffic back to old distribution
OLD_CF_DOMAIN="d123456old.cloudfront.net"

aws route53 change-resource-record-sets \
  --hosted-zone-id <ZONE_ID> \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "<app>.digistratum.com",
        "Type": "A",
        "AliasTarget": {
          "HostedZoneId": "Z2FDTNDATAQYW2",
          "DNSName": "'$OLD_CF_DOMAIN'",
          "EvaluateTargetHealth": true
        }
      }
    }]
  }'

# Wait for DNS propagation
sleep 60

# Verify
dig +short <app>.digistratum.com
# Should resolve to old CloudFront IPs
```

#### Re-Enable Stream Sync (Reverse Direction)

If writes occurred to new tables during cutover window, sync back:

```bash
# Deploy reverse sync Lambda (new → old)
cd ~/repos/digistratum/ds-app-template/tools
./deploy-reverse-sync-lambda.sh <app> prod

# This reads from new tables' streams and writes to old table
```

#### Verification

```bash
# Test old endpoints
curl -s https://<app>.digistratum.com/api/health
# Should return healthy from old stack

# Check Lambda logs (old)
aws logs tail /aws/lambda/<app>-prod --since 5m

# Verify no 5xx errors
aws cloudwatch get-metric-statistics \
  --namespace AWS/ApiGateway \
  --metric-name 5XXError \
  --dimensions Name=ApiName,Value=<app>-prod \
  --start-time $(date -u -v-1H +%Y-%m-%dT%H:%M:%SZ) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --period 300 \
  --statistics Sum
```

---

### Phase 4B: Late Rollback (Data Reconciliation Required)

**Scenario:** Cutover completed, significant writes to new tables, need to reverse.

**Duration:** 1-2 hours

#### Assess Data Divergence

```bash
# Check write activity on new tables
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name ConsumedWriteCapacityUnits \
  --dimensions Name=TableName,Value=<app>-prod-digistratum \
  --start-time <cutover-time> \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --period 3600 \
  --statistics Sum

# If writes occurred, export recent data
aws dynamodb export-table-to-point-in-time \
  --table-arn arn:aws:dynamodb:us-east-1:<ACCOUNT>:table/<app>-prod-digistratum \
  --s3-bucket ds-backups \
  --s3-prefix rollback/<app>/$(date +%Y%m%d-%H%M%S) \
  --export-time $(date -u +%Y-%m-%dT%H:%M:%SZ)
```

#### Data Reconciliation Options

**Option A: Accept data loss (minimal writes)**

If only a few writes occurred and they can be re-done:
```bash
# Switch DNS back (see Phase 4A)
# Notify affected users to retry their actions
```

**Option B: Merge new data back to old table**

```bash
# Export new table items written during cutover
cd ~/repos/digistratum/ds-app-template/tools
./export-delta.sh <app>-prod-digistratum --since <cutover-timestamp>

# Review and import to old table
./import-delta.sh <app>-prod --file delta-export.json --preview
./import-delta.sh <app>-prod --file delta-export.json --execute
```

**Option C: Point-in-time recovery**

If old table needs restoration:
```bash
aws dynamodb restore-table-to-point-in-time \
  --source-table-name <app>-prod \
  --target-table-name <app>-prod-restored \
  --restore-date-time <pre-cutover-timestamp>

# Verify restored data
aws dynamodb scan --table-name <app>-prod-restored --select COUNT

# If good, rename tables
aws dynamodb update-table --table-name <app>-prod \
  --deletion-protection-enabled false
aws dynamodb delete-table --table-name <app>-prod

# Note: DynamoDB doesn't support rename - create new table and import
```

---

### Phase 4C: Full Restore (Old Stack Deleted)

**Scenario:** Old stack (`<app>-prod`) was destroyed, need to fully restore.

**Duration:** 2-4 hours

**⚠️ Warning:** This is the most complex rollback scenario. Avoid by waiting 24-48 hours before cleanup.

#### Step 1: Restore CloudFormation Stack

```bash
# Restore from exported template
aws cloudformation create-stack \
  --stack-name <app>-prod \
  --template-body file://backup/<app>-prod-template.json \
  --capabilities CAPABILITY_NAMED_IAM \
  --parameters ParameterKey=Environment,ParameterValue=prod

aws cloudformation wait stack-create-complete --stack-name <app>-prod
```

#### Step 2: Restore DynamoDB Data

```bash
# If backup exists, restore
aws dynamodb restore-table-from-backup \
  --target-table-name <app>-prod \
  --backup-arn arn:aws:dynamodb:us-east-1:<ACCOUNT>:table/<app>-prod/backup/<backup-id>

# Or restore from S3 export
# (requires external tooling to import DynamoDB JSON format)
cd ~/repos/digistratum/ds-app-template/tools
./restore-from-export.sh <app>-prod s3://ds-backups/final-migration-backup/<app>-prod/
```

#### Step 3: Merge Post-Cutover Data

Data written to new tables after cutover needs merging:

```bash
# Export from new ecosystem tables
./export-delta.sh <app>-prod-digistratum --since <cutover-timestamp>
./export-delta.sh <app>-prod-leapkick --since <cutover-timestamp>

# Map ecosystem data back to single-table format
./transform-ecosystem-to-single.sh delta-digistratum.json delta-leapkick.json > merged-delta.json

# Import to restored table
./import-delta.sh <app>-prod --file merged-delta.json --execute
```

#### Step 4: Restore DNS

```bash
OLD_CF_DOMAIN=$(aws cloudformation describe-stacks \
  --stack-name <app>-prod \
  --query 'Stacks[0].Outputs[?OutputKey==`CloudFrontDomain`].OutputValue' --output text)

aws route53 change-resource-record-sets \
  --hosted-zone-id <ZONE_ID> \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "<app>.digistratum.com",
        "Type": "A",
        "AliasTarget": {
          "HostedZoneId": "Z2FDTNDATAQYW2",
          "DNSName": "'$OLD_CF_DOMAIN'",
          "EvaluateTargetHealth": true
        }
      }
    }]
  }'
```

#### Step 5: Cleanup Failed Migration Stacks

```bash
# After successful restore, remove multi-ecosystem stacks
cdk destroy <app>-app-prod --context env=prod --force
cdk destroy <app>-data-prod-leapkick --context env=prod --force
cdk destroy <app>-data-prod-digistratum --context env=prod --force
```

---

## When NOT to Rollback

### Data Divergence Too Large

If significant business data was written to new tables and cannot be cleanly merged, **do not rollback**. Instead:

1. Fix the issue in new stacks
2. Communicate with users about the transition
3. Accept that new architecture is now authoritative

### Irreversible Changes

Some changes cannot be easily reversed:
- User-generated content in new tables with new schema
- Third-party integrations reconfigured
- External systems updated to point to new endpoints

### Time-Sensitive Operations

During critical business periods (month-end, launches), **stabilize forward** rather than risk rollback disruption.

---

## Post-Rollback Checklist

After any rollback:

- [ ] Document root cause in issue comments
- [ ] Update migration plan with lessons learned
- [ ] Verify all monitoring/alerting restored
- [ ] Notify stakeholders of rollback
- [ ] Schedule retry with fixes applied
- [ ] Clean up any orphaned resources

---

## Related Documentation

- [Migration Guide](./migration-guide.md) — Full migration procedures
- [ECOSYSTEMS.md](./ECOSYSTEMS.md) — Configuration reference
- [CI_CD.md](./CI_CD.md) — Deployment pipeline
