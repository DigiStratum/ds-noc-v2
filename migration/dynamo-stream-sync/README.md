# DynamoDB Streams Migration Helper

CDK construct and Lambda function for syncing data from a source DynamoDB table to multiple ecosystem-specific target tables during multi-ecosystem migration.

## Overview

This helper enables Phase 3 of the multi-ecosystem migration: DynamoDB Streams-based data synchronization. It:

1. **Captures changes** from the source (legacy) table via DynamoDB Streams
2. **Transforms keys** if naming conventions change between old and new tables
3. **Replicates records** to all configured ecosystem-specific target tables
4. **Monitors sync lag** via CloudWatch dashboard

## Quick Start

### 1. Add to your CDK stack

```typescript
import { StreamSyncConstruct } from '../lib/migration';

// In your stack constructor:
new StreamSyncConstruct(this, 'StreamSync', {
  appName: 'myapp',
  envName: 'prod',
  sourceTable: legacyTable,  // Your existing table (must have streams enabled)
  targetTables: [
    { ecosystem: 'digistratum', table: dsDataStack.table },
    { ecosystem: 'leapkick', table: lkDataStack.table },
  ],
  keyTransform: { type: 'prefix' },  // Optional: transform keys
  enabled: true,
});
```

### 2. Enable DynamoDB Streams on source table

If your source table doesn't have streams enabled, you'll need to enable them:

```typescript
// For a new table:
const table = new dynamodb.Table(this, 'Table', {
  // ... other props
  stream: dynamodb.StreamViewType.NEW_AND_OLD_IMAGES,
});

// For an existing table, enable via AWS Console or CLI:
// aws dynamodb update-table --table-name MyTable \
//   --stream-specification StreamEnabled=true,StreamViewType=NEW_AND_OLD_IMAGES
```

### 3. Deploy and monitor

```bash
cd infra
npx cdk deploy

# Monitor via CloudWatch Dashboard (URL in stack outputs)
```

## Configuration

### StreamSyncProps

| Property | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `appName` | string | ✅ | - | Application name for resource naming |
| `envName` | string | ✅ | - | Environment (dev/prod) |
| `sourceTable` | ITable | ✅ | - | Source DynamoDB table |
| `targetTables` | TargetTableConfig[] | ✅ | - | Target ecosystem tables |
| `keyTransform` | KeyTransformConfig | ❌ | `{ type: 'none' }` | Key transformation config |
| `enabled` | boolean | ❌ | `true` | Initial enabled state |
| `memorySize` | number | ❌ | `256` | Lambda memory (MB) |
| `timeout` | number | ❌ | `60` | Lambda timeout (seconds) |
| `batchSize` | number | ❌ | `100` | Max records per batch |
| `maxBatchingWindow` | number | ❌ | `5` | Max batching window (seconds) |
| `startingPosition` | StartingPosition | ❌ | `TRIM_HORIZON` | Where to start processing |
| `enableDashboard` | boolean | ❌ | `true` | Create CloudWatch dashboard |

### Key Transform Types

| Type | Description | Example |
|------|-------------|---------|
| `none` | No transformation (direct copy) | `USER#123` → `USER#123` |
| `prefix` | Add ecosystem prefix | `USER#123` → `digistratum#USER#123` |
| `suffix` | Add ecosystem suffix | `USER#123` → `USER#123#digistratum` |
| `custom` | Custom transform function | See Lambda code |

## Runtime Control

### Toggle sync on/off without redeployment

The construct creates an SSM Parameter that controls whether sync is active:

```bash
# Disable sync
aws ssm put-parameter \
  --name "/myapp/prod/stream-sync/enabled" \
  --value "false" \
  --overwrite

# Enable sync
aws ssm put-parameter \
  --name "/myapp/prod/stream-sync/enabled" \
  --value "true" \
  --overwrite
```

The Lambda checks this parameter every 30 seconds. When disabled, records are received but not processed (no DynamoDB writes to target tables).

## Monitoring

### CloudWatch Dashboard

The construct creates a dashboard with:

1. **Invocations & Errors** - Lambda execution metrics
2. **Processing Duration** - How long each batch takes
3. **Iterator Age (Sync Lag)** - How far behind real-time the processor is
4. **Records Processed/Skipped/Failed** - Custom metrics from Lambda
5. **Target Table Writes** - Write capacity consumed per ecosystem

### Key Metrics

| Metric | Namespace | Description |
|--------|-----------|-------------|
| `IteratorAge` | AWS/Lambda | Sync lag in milliseconds |
| `RecordsProcessed` | {app}-{env}-stream-sync/Metrics | Successfully synced records |
| `RecordsSkipped` | {app}-{env}-stream-sync/Metrics | Records skipped (disabled) |
| `RecordsFailed` | {app}-{env}-stream-sync/Metrics | Records that failed processing |

### Alerting

An alarm is created for sync lag > 60 seconds:

```
Alarm: {appName}-{envName}-stream-sync-SyncLagAlarm
Condition: IteratorAge > 60000ms for 3 consecutive periods
```

## Migration Workflow

### Phase 3: Enable Sync

1. **Ensure source table has streams enabled** (NEW_AND_OLD_IMAGES)
2. **Deploy StreamSyncConstruct** with `enabled: true`
3. **Verify via dashboard** that records are syncing

### Phase 4: Cutover

1. **Verify sync is caught up** (Iterator Age near 0)
2. **Switch application to new tables**
3. **Disable sync** via SSM parameter (not immediate teardown)
4. **Monitor for any sync-back requirements**

### Phase 5: Cleanup

1. **Remove StreamSyncConstruct** from CDK
2. **Delete SSM parameter** if not auto-cleaned
3. **Disable streams** on source table if no longer needed

## Custom Transforms

For complex key transformations, add custom functions to the Lambda:

```typescript
// In lambda/index.ts, customTransforms object:
const customTransforms: Record<string, TransformFn> = {
  myCustomTransform: (key, ecosystem) => {
    // Your transformation logic
    return `${ecosystem}:${key.toUpperCase()}`;
  },
};
```

Then configure:

```typescript
new StreamSyncConstruct(this, 'StreamSync', {
  // ...
  keyTransform: {
    type: 'custom',
    customTransformName: 'myCustomTransform',
  },
});
```

## Troubleshooting

### High Iterator Age

- Increase Lambda memory/timeout
- Decrease batch size
- Check for errors in CloudWatch Logs

### Records Not Syncing

- Verify source table has streams enabled with NEW_AND_OLD_IMAGES
- Check SSM parameter is set to "true"
- Verify Lambda has IAM permissions for target tables

### Duplicate Records

- Idempotent by design: INSERT/MODIFY use PutItem (overwrites)
- If you see unexpected duplicates, check key transformation logic

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────────┐
│   Source    │────▶│  DynamoDB   │────▶│  Stream Sync    │
│   Table     │     │   Streams   │     │    Lambda       │
└─────────────┘     └─────────────┘     └────────┬────────┘
                                                 │
                    ┌────────────────────────────┼────────────────────────────┐
                    │                            │                            │
                    ▼                            ▼                            ▼
          ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
          │  digistratum    │         │   leapkick      │         │   (other eco)   │
          │     table       │         │     table       │         │     table       │
          └─────────────────┘         └─────────────────┘         └─────────────────┘
```

## Files

```
ds-app-template/
├── infra/lib/migration/
│   ├── index.ts                    # Barrel export (CDK constructs)
│   └── stream-sync-construct.ts    # CDK construct
├── migration/dynamo-stream-sync/
│   ├── lambda/
│   │   ├── index.ts               # Lambda handler
│   │   ├── package.json           # Lambda dependencies
│   │   └── dist/                  # Built Lambda (npm run build)
│   └── README.md                  # This file
```

## Dependencies

**CDK Construct:**
- aws-cdk-lib (dynamodb, lambda, ssm, cloudwatch, logs, iam)
- constructs

**Lambda:**
- @aws-sdk/client-dynamodb
- @aws-sdk/lib-dynamodb
- @aws-sdk/client-ssm
- @aws-sdk/client-cloudwatch
