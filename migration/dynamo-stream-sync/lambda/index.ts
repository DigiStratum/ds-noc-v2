/**
 * DynamoDB Streams Sync Processor Lambda
 *
 * Processes DynamoDB Stream events and replicates records to target
 * ecosystem-specific tables. Supports key transformation for naming
 * convention changes during multi-ecosystem migration.
 *
 * Environment variables:
 * - SOURCE_TABLE_NAME: Name of source table (for logging)
 * - TARGET_TABLES_CONFIG: JSON map of ecosystem -> { tableName, tableArn }
 * - KEY_TRANSFORM_TYPE: 'prefix' | 'suffix' | 'custom' | 'none'
 * - KEY_TRANSFORM_DELIMITER: Delimiter for prefix/suffix (default: '#')
 * - KEY_TRANSFORM_CUSTOM_NAME: Name of custom transform function
 * - ENABLED_PARAMETER_NAME: SSM parameter name for enable/disable toggle
 * - APP_NAME: Application name (for metrics namespace)
 * - ENV_NAME: Environment name (for metrics namespace)
 */

import { DynamoDBStreamEvent, DynamoDBRecord, StreamRecord, Context } from 'aws-lambda';
import { DynamoDBClient } from '@aws-sdk/client-dynamodb';
import {
  DynamoDBDocumentClient,
  PutCommand,
  DeleteCommand,
} from '@aws-sdk/lib-dynamodb';
import { SSMClient, GetParameterCommand } from '@aws-sdk/client-ssm';
import {
  CloudWatchClient,
  PutMetricDataCommand,
  MetricDatum,
  StandardUnit,
} from '@aws-sdk/client-cloudwatch';
import { unmarshall } from '@aws-sdk/util-dynamodb';
import { AttributeValue } from '@aws-sdk/client-dynamodb';

// -----------------------------------------------------------------------------
// Configuration
// -----------------------------------------------------------------------------

interface TargetTableConfig {
  tableName: string;
  tableArn: string;
}

const config = {
  sourceTableName: process.env.SOURCE_TABLE_NAME || '',
  targetTables: JSON.parse(
    process.env.TARGET_TABLES_CONFIG || '{}'
  ) as Record<string, TargetTableConfig>,
  keyTransform: {
    type: (process.env.KEY_TRANSFORM_TYPE || 'none') as
      | 'prefix'
      | 'suffix'
      | 'custom'
      | 'none',
    delimiter: process.env.KEY_TRANSFORM_DELIMITER || '#',
    customName: process.env.KEY_TRANSFORM_CUSTOM_NAME || '',
  },
  enabledParameterName: process.env.ENABLED_PARAMETER_NAME || '',
  appName: process.env.APP_NAME || 'app',
  envName: process.env.ENV_NAME || 'dev',
};

// Metrics namespace
const metricsNamespace = `${config.appName}-${config.envName}-stream-sync/Metrics`;

// -----------------------------------------------------------------------------
// AWS Clients
// -----------------------------------------------------------------------------

const dynamoClient = new DynamoDBClient({});
const docClient = DynamoDBDocumentClient.from(dynamoClient, {
  marshallOptions: {
    removeUndefinedValues: true,
  },
});

const ssmClient = new SSMClient({});
const cloudwatchClient = new CloudWatchClient({});

// Cache for enabled state (refreshed periodically)
let enabledCache: { value: boolean; timestamp: number } | null = null;
const CACHE_TTL_MS = 30_000; // 30 seconds

// -----------------------------------------------------------------------------
// Key Transform Functions
// -----------------------------------------------------------------------------

type TransformFn = (key: string, ecosystem: string) => string;

/**
 * Built-in transform: Add ecosystem prefix to key
 * Example: "USER#123" -> "digistratum#USER#123"
 */
const prefixTransform: TransformFn = (key, ecosystem) => {
  return `${ecosystem}${config.keyTransform.delimiter}${key}`;
};

/**
 * Built-in transform: Add ecosystem suffix to key
 * Example: "USER#123" -> "USER#123#digistratum"
 */
const suffixTransform: TransformFn = (key, ecosystem) => {
  return `${key}${config.keyTransform.delimiter}${ecosystem}`;
};

/**
 * No-op transform: Return key unchanged
 */
const noTransform: TransformFn = (key) => key;

/**
 * Custom transform registry
 * Add custom transform functions here for specific migration needs
 */
const customTransforms: Record<string, TransformFn> = {
  // Example: Convert PK format from "TYPE#ID" to "ECOSYSTEM:TYPE#ID"
  colonPrefix: (key, ecosystem) => `${ecosystem}:${key}`,

  // Example: Extract and re-prefix tenant IDs
  tenantRewrite: (key, ecosystem) => {
    // Original: "TENANT#old-tenant#USER#123"
    // New: "TENANT#ecosystem#USER#123"
    if (key.startsWith('TENANT#')) {
      const parts = key.split('#');
      if (parts.length >= 4) {
        parts[1] = ecosystem;
        return parts.join('#');
      }
    }
    return key;
  },
};

/**
 * Get the appropriate transform function based on config
 */
function getTransformFn(): TransformFn {
  switch (config.keyTransform.type) {
    case 'prefix':
      return prefixTransform;
    case 'suffix':
      return suffixTransform;
    case 'custom':
      const customFn = customTransforms[config.keyTransform.customName];
      if (!customFn) {
        console.warn(
          `Custom transform '${config.keyTransform.customName}' not found, using no-op`
        );
        return noTransform;
      }
      return customFn;
    case 'none':
    default:
      return noTransform;
  }
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

/**
 * Check if sync is enabled via SSM Parameter
 */
async function isSyncEnabled(): Promise<boolean> {
  // Check cache first
  if (enabledCache && Date.now() - enabledCache.timestamp < CACHE_TTL_MS) {
    return enabledCache.value;
  }

  try {
    const response = await ssmClient.send(
      new GetParameterCommand({
        Name: config.enabledParameterName,
      })
    );

    const enabled = response.Parameter?.Value?.toLowerCase() === 'true';
    enabledCache = { value: enabled, timestamp: Date.now() };
    return enabled;
  } catch (error) {
    console.error('Error fetching enabled parameter:', error);
    // Default to enabled if we can't read the parameter
    return true;
  }
}

/**
 * Transform a DynamoDB item for the target ecosystem
 */
function transformItem(
  item: Record<string, unknown>,
  ecosystem: string,
  transformFn: TransformFn
): Record<string, unknown> {
  const transformed = { ...item };

  // Transform PK
  if (typeof item.PK === 'string') {
    transformed.PK = transformFn(item.PK, ecosystem);
  }

  // Transform SK
  if (typeof item.SK === 'string') {
    transformed.SK = transformFn(item.SK, ecosystem);
  }

  // Transform GSI keys if present
  if (typeof item.GSI1PK === 'string') {
    transformed.GSI1PK = transformFn(item.GSI1PK, ecosystem);
  }
  if (typeof item.GSI1SK === 'string') {
    transformed.GSI1SK = transformFn(item.GSI1SK, ecosystem);
  }

  return transformed;
}

/**
 * Process a single DynamoDB Stream record
 */
async function processRecord(
  record: DynamoDBRecord,
  ecosystem: string,
  targetTable: string,
  transformFn: TransformFn
): Promise<{ success: boolean; eventName: string }> {
  const eventName = record.eventName || 'UNKNOWN';

  try {
    const streamRecord = record.dynamodb;
    if (!streamRecord) {
      console.warn('No dynamodb property in record');
      return { success: false, eventName };
    }

    switch (eventName) {
      case 'INSERT':
      case 'MODIFY': {
        // Get the new image
        const newImage = streamRecord.NewImage;
        if (!newImage) {
          console.warn('No NewImage for INSERT/MODIFY');
          return { success: false, eventName };
        }

        // Unmarshall and transform
        const item = unmarshall(newImage as Record<string, AttributeValue>);
        const transformedItem = transformItem(item, ecosystem, transformFn);

        // Write to target table
        await docClient.send(
          new PutCommand({
            TableName: targetTable,
            Item: transformedItem,
          })
        );

        console.log(`[${eventName}] Replicated to ${ecosystem}:`, transformedItem.PK);
        return { success: true, eventName };
      }

      case 'REMOVE': {
        // Get the keys from old image
        const keys = streamRecord.Keys;
        if (!keys) {
          console.warn('No Keys for REMOVE');
          return { success: false, eventName };
        }

        const keyItem = unmarshall(keys as Record<string, AttributeValue>);
        const transformedKeys: Record<string, string> = {};

        if (typeof keyItem.PK === 'string') {
          transformedKeys.PK = transformFn(keyItem.PK, ecosystem);
        }
        if (typeof keyItem.SK === 'string') {
          transformedKeys.SK = transformFn(keyItem.SK, ecosystem);
        }

        // Delete from target table
        await docClient.send(
          new DeleteCommand({
            TableName: targetTable,
            Key: transformedKeys,
          })
        );

        console.log(`[REMOVE] Deleted from ${ecosystem}:`, transformedKeys.PK);
        return { success: true, eventName };
      }

      default:
        console.warn(`Unknown event type: ${eventName}`);
        return { success: false, eventName };
    }
  } catch (error) {
    console.error(`Error processing ${eventName} for ${ecosystem}:`, error);
    return { success: false, eventName };
  }
}

/**
 * Publish metrics to CloudWatch
 */
async function publishMetrics(
  processed: number,
  skipped: number,
  failed: number
): Promise<void> {
  const metrics: MetricDatum[] = [
    {
      MetricName: 'RecordsProcessed',
      Value: processed,
      Unit: StandardUnit.Count,
    },
    {
      MetricName: 'RecordsSkipped',
      Value: skipped,
      Unit: StandardUnit.Count,
    },
    {
      MetricName: 'RecordsFailed',
      Value: failed,
      Unit: StandardUnit.Count,
    },
  ];

  try {
    await cloudwatchClient.send(
      new PutMetricDataCommand({
        Namespace: metricsNamespace,
        MetricData: metrics,
      })
    );
  } catch (error) {
    console.error('Error publishing metrics:', error);
  }
}

// -----------------------------------------------------------------------------
// Lambda Handler
// -----------------------------------------------------------------------------

interface BatchItemFailure {
  itemIdentifier: string;
}

interface StreamBatchResponse {
  batchItemFailures: BatchItemFailure[];
}

export async function handler(
  event: DynamoDBStreamEvent,
  context: Context
): Promise<StreamBatchResponse> {
  console.log(`Processing ${event.Records.length} records`);

  // Check if sync is enabled
  const enabled = await isSyncEnabled();
  if (!enabled) {
    console.log('Sync is disabled via SSM parameter, skipping all records');
    await publishMetrics(0, event.Records.length, 0);
    return { batchItemFailures: [] };
  }

  const transformFn = getTransformFn();
  const ecosystems = Object.keys(config.targetTables);
  const batchItemFailures: BatchItemFailure[] = [];

  let processed = 0;
  let skipped = 0;
  let failed = 0;

  // Process each record
  for (const record of event.Records) {
    const recordId = record.eventID;
    if (!recordId) {
      skipped++;
      continue;
    }

    let recordFailed = false;

    // Replicate to all target ecosystems
    for (const ecosystem of ecosystems) {
      const targetTable = config.targetTables[ecosystem].tableName;

      const result = await processRecord(record, ecosystem, targetTable, transformFn);

      if (result.success) {
        processed++;
      } else {
        recordFailed = true;
        failed++;
      }
    }

    // Report partial failures for retry
    if (recordFailed) {
      batchItemFailures.push({ itemIdentifier: recordId });
    }
  }

  console.log(`Processed: ${processed}, Skipped: ${skipped}, Failed: ${failed}`);
  await publishMetrics(processed, skipped, failed);

  return { batchItemFailures };
}
