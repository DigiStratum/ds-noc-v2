/**
 * DynamoDB Streams Sync Construct
 *
 * CDK construct for syncing data from a source DynamoDB table to target
 * ecosystem-specific tables during multi-ecosystem migration.
 *
 * Features:
 * - Enables DynamoDB Streams on source table
 * - Lambda function to replicate INSERT/MODIFY/DELETE events
 * - Optional key transformation for naming convention changes
 * - Runtime enable/disable toggle via SSM Parameter
 * - CloudWatch dashboard for sync lag monitoring
 */
import * as cdk from 'aws-cdk-lib';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as lambdaEventSources from 'aws-cdk-lib/aws-lambda-event-sources';
import * as ssm from 'aws-cdk-lib/aws-ssm';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as logs from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';
import * as path from 'path';

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

/**
 * Configuration for a target ecosystem table
 */
export interface TargetTableConfig {
  /**
   * Ecosystem identifier (e.g., 'digistratum', 'leapkick')
   */
  ecosystem: string;

  /**
   * DynamoDB table for this ecosystem
   */
  table: dynamodb.ITable;
}

/**
 * Key transformation configuration
 */
export interface KeyTransformConfig {
  /**
   * Transform function name - the Lambda will apply this transformation
   * Options:
   * - 'prefix': Add ecosystem prefix to PK/SK (e.g., PK="USER#123" -> PK="digistratum#USER#123")
   * - 'suffix': Add ecosystem suffix to PK/SK
   * - 'custom': Use custom transformation defined in Lambda code
   * - 'none': No transformation (direct copy)
   */
  type: 'prefix' | 'suffix' | 'custom' | 'none';

  /**
   * Delimiter for prefix/suffix (default: '#')
   */
  delimiter?: string;

  /**
   * For 'custom' type: name of the custom transform function in Lambda
   */
  customTransformName?: string;
}

/**
 * Props for StreamSyncConstruct
 */
export interface StreamSyncProps {
  /**
   * Application name for resource naming
   */
  appName: string;

  /**
   * Environment name (dev/prod)
   */
  envName: string;

  /**
   * Source DynamoDB table (streams will be enabled on this table)
   * Note: The table must have streams enabled or be able to enable them
   */
  sourceTable: dynamodb.ITable;

  /**
   * Target tables mapped by ecosystem name
   */
  targetTables: TargetTableConfig[];

  /**
   * Optional key transformation configuration
   * @default { type: 'none' }
   */
  keyTransform?: KeyTransformConfig;

  /**
   * Initial enabled state for sync
   * Can be toggled at runtime via SSM Parameter
   * @default true
   */
  enabled?: boolean;

  /**
   * Lambda memory size in MB
   * @default 256
   */
  memorySize?: number;

  /**
   * Lambda timeout in seconds
   * @default 60
   */
  timeout?: number;

  /**
   * Maximum batch size for stream records
   * @default 100
   */
  batchSize?: number;

  /**
   * Maximum batching window in seconds
   * @default 5
   */
  maxBatchingWindow?: number;

  /**
   * Starting position for stream processing
   * @default TRIM_HORIZON (process all existing records)
   */
  startingPosition?: lambda.StartingPosition;

  /**
   * Enable detailed CloudWatch dashboard
   * @default true
   */
  enableDashboard?: boolean;
}

// -----------------------------------------------------------------------------
// StreamSyncConstruct
// -----------------------------------------------------------------------------

export class StreamSyncConstruct extends Construct {
  /**
   * The stream processor Lambda function
   */
  public readonly processorFunction: lambda.Function;

  /**
   * SSM Parameter for runtime enable/disable toggle
   */
  public readonly enabledParameter: ssm.StringParameter;

  /**
   * CloudWatch Dashboard for sync monitoring (if enabled)
   */
  public readonly dashboard?: cloudwatch.Dashboard;

  /**
   * Log group for the processor Lambda
   */
  public readonly logGroup: logs.LogGroup;

  constructor(scope: Construct, id: string, props: StreamSyncProps) {
    super(scope, id);

    const {
      appName,
      envName,
      sourceTable,
      targetTables,
      keyTransform = { type: 'none' },
      enabled = true,
      memorySize = 256,
      timeout = 60,
      batchSize = 100,
      maxBatchingWindow = 5,
      startingPosition = lambda.StartingPosition.TRIM_HORIZON,
      enableDashboard = true,
    } = props;

    const resourcePrefix = `${appName}-${envName}-stream-sync`;

    // Validate
    if (targetTables.length === 0) {
      throw new Error('At least one target table is required');
    }

    // ---------------------------------------------------------------------
    // SSM Parameter for runtime toggle
    // ---------------------------------------------------------------------
    this.enabledParameter = new ssm.StringParameter(this, 'EnabledParameter', {
      parameterName: `/${appName}/${envName}/stream-sync/enabled`,
      stringValue: enabled ? 'true' : 'false',
      description: 'Toggle DynamoDB Streams sync on/off without redeployment',
      tier: ssm.ParameterTier.STANDARD,
    });

    // ---------------------------------------------------------------------
    // Log Group
    // ---------------------------------------------------------------------
    this.logGroup = new logs.LogGroup(this, 'LogGroup', {
      logGroupName: `/aws/lambda/${resourcePrefix}`,
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // ---------------------------------------------------------------------
    // Build target tables config for Lambda environment
    // ---------------------------------------------------------------------
    const targetTablesConfig: Record<string, { tableName: string; tableArn: string }> = {};
    for (const target of targetTables) {
      targetTablesConfig[target.ecosystem] = {
        tableName: target.table.tableName,
        tableArn: target.table.tableArn,
      };
    }

    // ---------------------------------------------------------------------
    // Lambda Function
    // Lambda code is in migration/dynamo-stream-sync/lambda/dist/
    // Must run `npm install && npm run build` in lambda folder first
    // ---------------------------------------------------------------------
    const lambdaCodePath = path.join(__dirname, '..', '..', '..', 'migration', 'dynamo-stream-sync', 'lambda', 'dist');
    
    this.processorFunction = new lambda.Function(this, 'ProcessorFunction', {
      functionName: resourcePrefix,
      runtime: lambda.Runtime.NODEJS_20_X,
      architecture: lambda.Architecture.ARM_64,
      handler: 'index.handler',
      code: lambda.Code.fromAsset(lambdaCodePath),
      memorySize,
      timeout: cdk.Duration.seconds(timeout),
      logGroup: this.logGroup,
      environment: {
        SOURCE_TABLE_NAME: sourceTable.tableName,
        TARGET_TABLES_CONFIG: JSON.stringify(targetTablesConfig),
        KEY_TRANSFORM_TYPE: keyTransform.type,
        KEY_TRANSFORM_DELIMITER: keyTransform.delimiter || '#',
        KEY_TRANSFORM_CUSTOM_NAME: keyTransform.customTransformName || '',
        ENABLED_PARAMETER_NAME: this.enabledParameter.parameterName,
        APP_NAME: appName,
        ENV_NAME: envName,
      },
    });

    // Grant read access to the SSM parameter
    this.enabledParameter.grantRead(this.processorFunction);

    // Grant read/write to all target tables
    for (const target of targetTables) {
      target.table.grantReadWriteData(this.processorFunction);
    }

    // Grant read from source table (for verification/retries)
    sourceTable.grantReadData(this.processorFunction);

    // ---------------------------------------------------------------------
    // DynamoDB Streams Event Source
    // ---------------------------------------------------------------------
    this.processorFunction.addEventSource(
      new lambdaEventSources.DynamoEventSource(sourceTable, {
        startingPosition,
        batchSize,
        maxBatchingWindow: cdk.Duration.seconds(maxBatchingWindow),
        retryAttempts: 3,
        bisectBatchOnError: true,
        reportBatchItemFailures: true,
        enabled: true, // Event source is always enabled; Lambda checks SSM param
      })
    );

    // ---------------------------------------------------------------------
    // CloudWatch Dashboard
    // ---------------------------------------------------------------------
    if (enableDashboard) {
      this.dashboard = this.createDashboard(
        resourcePrefix,
        sourceTable,
        targetTables
      );
    }

    // ---------------------------------------------------------------------
    // Outputs
    // ---------------------------------------------------------------------
    new cdk.CfnOutput(this, 'ProcessorFunctionArn', {
      value: this.processorFunction.functionArn,
      description: 'Stream processor Lambda ARN',
    });

    new cdk.CfnOutput(this, 'EnabledParameterName', {
      value: this.enabledParameter.parameterName,
      description: 'SSM Parameter to toggle sync (true/false)',
    });

    new cdk.CfnOutput(this, 'ToggleCommand', {
      value: `aws ssm put-parameter --name "${this.enabledParameter.parameterName}" --value "false" --overwrite`,
      description: 'Command to disable sync',
    });

    if (this.dashboard) {
      new cdk.CfnOutput(this, 'DashboardUrl', {
        value: `https://${cdk.Stack.of(this).region}.console.aws.amazon.com/cloudwatch/home?region=${cdk.Stack.of(this).region}#dashboards:name=${resourcePrefix}`,
        description: 'CloudWatch Dashboard URL',
      });
    }
  }

  /**
   * Create CloudWatch Dashboard for monitoring sync operations
   */
  private createDashboard(
    resourcePrefix: string,
    sourceTable: dynamodb.ITable,
    targetTables: TargetTableConfig[]
  ): cloudwatch.Dashboard {
    const dashboard = new cloudwatch.Dashboard(this, 'Dashboard', {
      dashboardName: resourcePrefix,
    });

    // Lambda invocation metrics
    const invocationsMetric = this.processorFunction.metricInvocations({
      period: cdk.Duration.minutes(1),
    });

    const errorsMetric = this.processorFunction.metricErrors({
      period: cdk.Duration.minutes(1),
    });

    const durationMetric = this.processorFunction.metricDuration({
      period: cdk.Duration.minutes(1),
    });

    const throttlesMetric = this.processorFunction.metricThrottles({
      period: cdk.Duration.minutes(1),
    });

    // Row 1: Lambda invocation stats
    dashboard.addWidgets(
      new cloudwatch.GraphWidget({
        title: 'Stream Processor Invocations',
        left: [invocationsMetric],
        right: [errorsMetric],
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'Processing Duration (ms)',
        left: [durationMetric],
        width: 6,
      }),
      new cloudwatch.GraphWidget({
        title: 'Throttles',
        left: [throttlesMetric],
        width: 6,
      })
    );

    // Row 2: Iterator age (sync lag indicator)
    const iteratorAgeMetric = new cloudwatch.Metric({
      namespace: 'AWS/Lambda',
      metricName: 'IteratorAge',
      dimensionsMap: {
        FunctionName: this.processorFunction.functionName,
      },
      statistic: 'Maximum',
      period: cdk.Duration.minutes(1),
    });

    dashboard.addWidgets(
      new cloudwatch.GraphWidget({
        title: 'Iterator Age (Sync Lag) - Lower is Better',
        left: [iteratorAgeMetric],
        width: 12,
      }),
      new cloudwatch.SingleValueWidget({
        title: 'Current Sync Lag',
        metrics: [iteratorAgeMetric],
        width: 6,
      }),
      new cloudwatch.AlarmWidget({
        title: 'Sync Lag Alarm',
        alarm: new cloudwatch.Alarm(this, 'SyncLagAlarm', {
          metric: iteratorAgeMetric,
          threshold: 60000, // 60 seconds
          evaluationPeriods: 3,
          alarmDescription: 'Stream sync is falling behind (>60s lag)',
        }),
        width: 6,
      })
    );

    // Row 3: Records processed (custom metrics from Lambda)
    const recordsProcessedMetric = new cloudwatch.Metric({
      namespace: `${resourcePrefix}/Metrics`,
      metricName: 'RecordsProcessed',
      statistic: 'Sum',
      period: cdk.Duration.minutes(1),
    });

    const recordsSkippedMetric = new cloudwatch.Metric({
      namespace: `${resourcePrefix}/Metrics`,
      metricName: 'RecordsSkipped',
      statistic: 'Sum',
      period: cdk.Duration.minutes(1),
    });

    const recordsFailedMetric = new cloudwatch.Metric({
      namespace: `${resourcePrefix}/Metrics`,
      metricName: 'RecordsFailed',
      statistic: 'Sum',
      period: cdk.Duration.minutes(1),
    });

    dashboard.addWidgets(
      new cloudwatch.GraphWidget({
        title: 'Records Processed',
        left: [recordsProcessedMetric, recordsSkippedMetric],
        right: [recordsFailedMetric],
        width: 12,
      }),
      new cloudwatch.SingleValueWidget({
        title: 'Total Processed (24h)',
        metrics: [
          new cloudwatch.Metric({
            namespace: `${resourcePrefix}/Metrics`,
            metricName: 'RecordsProcessed',
            statistic: 'Sum',
            period: cdk.Duration.hours(24),
          }),
        ],
        width: 6,
      }),
      new cloudwatch.SingleValueWidget({
        title: 'Total Failed (24h)',
        metrics: [
          new cloudwatch.Metric({
            namespace: `${resourcePrefix}/Metrics`,
            metricName: 'RecordsFailed',
            statistic: 'Sum',
            period: cdk.Duration.hours(24),
          }),
        ],
        width: 6,
      })
    );

    // Row 4: Target table write metrics
    const targetWidgets: cloudwatch.IWidget[] = [];
    for (const target of targetTables) {
      targetWidgets.push(
        new cloudwatch.GraphWidget({
          title: `${target.ecosystem} Table Writes`,
          left: [
            new cloudwatch.Metric({
              namespace: 'AWS/DynamoDB',
              metricName: 'ConsumedWriteCapacityUnits',
              dimensionsMap: {
                TableName: target.table.tableName,
              },
              statistic: 'Sum',
              period: cdk.Duration.minutes(1),
            }),
          ],
          width: Math.floor(24 / targetTables.length),
        })
      );
    }
    dashboard.addWidgets(...targetWidgets);

    return dashboard;
  }
}
