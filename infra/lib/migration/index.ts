/**
 * DynamoDB Streams Sync Migration Helper
 *
 * CDK constructs and Lambda for syncing data between DynamoDB tables
 * during multi-ecosystem migration.
 *
 * @example
 * ```typescript
 * import { StreamSyncConstruct } from './migration';
 *
 * new StreamSyncConstruct(this, 'StreamSync', {
 *   appName: 'myapp',
 *   envName: 'prod',
 *   sourceTable: legacyTable,
 *   targetTables: [
 *     { ecosystem: 'digistratum', table: dsTable },
 *     { ecosystem: 'leapkick', table: lkTable },
 *   ],
 *   keyTransform: { type: 'prefix' },
 *   enabled: true,
 * });
 * ```
 * 
 * Before deploying, build the Lambda:
 * ```bash
 * cd migration/dynamo-stream-sync/lambda && npm install && npm run build
 * ```
 */

export {
  StreamSyncConstruct,
  StreamSyncProps,
  TargetTableConfig,
  KeyTransformConfig,
} from './stream-sync-construct';
