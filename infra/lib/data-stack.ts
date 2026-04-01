/**
 * DS Data Stack - Per-ecosystem-env data isolation
 * Creates DynamoDB table and S3 bucket for each ecosystem-environment combination
 */
import * as cdk from 'aws-cdk-lib';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as s3 from 'aws-cdk-lib/aws-s3';
import { Construct } from 'constructs';

export interface DataStackProps extends cdk.StackProps {
  /**
   * Application name (e.g., 'bogusapp')
   */
  appName: string;
  
  /**
   * Environment name (e.g., 'dev', 'prod')
   */
  envName: string;
  
  /**
   * Ecosystem identifier (e.g., 'digistratum', 'leapkick')
   */
  ecosystem: string;
}

export class DataStack extends cdk.Stack {
  /**
   * The DynamoDB table for this ecosystem-env
   */
  public readonly table: dynamodb.Table;
  
  /**
   * The S3 bucket for assets in this ecosystem-env
   */
  public readonly assetsBucket: s3.Bucket;
  
  /**
   * Table name for cross-stack references
   */
  public readonly tableName: string;
  
  /**
   * Bucket name for cross-stack references
   */
  public readonly bucketName: string;
  
  constructor(scope: Construct, id: string, props: DataStackProps) {
    super(scope, id, {
      ...props,
      description: `${props.appName} data (${props.envName}/${props.ecosystem})`,
    });

    const { appName, envName, ecosystem } = props;
    const isProd = envName === 'prod';
    
    // Resource naming: {appName}-{envName}-{ecosystem}
    const resourcePrefix = `${appName}-${envName}-${ecosystem}`;
    this.tableName = resourcePrefix;
    this.bucketName = `${resourcePrefix}-assets`;

    // Apply standard tags
    cdk.Tags.of(this).add('Application', appName);
    cdk.Tags.of(this).add('Environment', envName);
    cdk.Tags.of(this).add('Ecosystem', ecosystem);
    cdk.Tags.of(this).add('ManagedBy', 'CDK');

    // DynamoDB Table with PK/SK and GSI1
    this.table = new dynamodb.Table(this, 'Table', {
      tableName: this.tableName,
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: isProd ? cdk.RemovalPolicy.RETAIN : cdk.RemovalPolicy.DESTROY,
      timeToLiveAttribute: 'TTL',
    });

    // Add GSI1 for common query patterns
    this.table.addGlobalSecondaryIndex({
      indexName: 'GSI1',
      partitionKey: { name: 'GSI1PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'GSI1SK', type: dynamodb.AttributeType.STRING },
    });

    // S3 Bucket with OAC-ready configuration
    this.assetsBucket = new s3.Bucket(this, 'AssetsBucket', {
      bucketName: `${this.bucketName}-${this.account}`,
      removalPolicy: isProd ? cdk.RemovalPolicy.RETAIN : cdk.RemovalPolicy.DESTROY,
      autoDeleteObjects: !isProd,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
      enforceSSL: true,
      versioned: isProd, // Enable versioning for prod
    });

    // CDK Outputs for cross-stack references
    new cdk.CfnOutput(this, 'TableName', {
      value: this.table.tableName,
      description: `DynamoDB table for ${ecosystem}`,
      exportName: `${resourcePrefix}-table-name`,
    });

    new cdk.CfnOutput(this, 'TableArn', {
      value: this.table.tableArn,
      description: `DynamoDB table ARN for ${ecosystem}`,
      exportName: `${resourcePrefix}-table-arn`,
    });

    new cdk.CfnOutput(this, 'BucketName', {
      value: this.assetsBucket.bucketName,
      description: `S3 bucket for ${ecosystem} assets`,
      exportName: `${resourcePrefix}-bucket-name`,
    });

    new cdk.CfnOutput(this, 'BucketArn', {
      value: this.assetsBucket.bucketArn,
      description: `S3 bucket ARN for ${ecosystem} assets`,
      exportName: `${resourcePrefix}-bucket-arn`,
    });
  }
}
