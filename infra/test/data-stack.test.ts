import * as cdk from 'aws-cdk-lib';
import { Template, Match } from 'aws-cdk-lib/assertions';
import { DataStack } from '../lib/data-stack';

describe('DataStack', () => {
  describe('dev environment', () => {
    let stack: DataStack;
    let template: Template;

    beforeAll(() => {
      const app = new cdk.App();
      stack = new DataStack(app, 'TestDataStackDev', {
        appName: 'testapp',
        envName: 'dev',
        ecosystem: 'digistratum',
        env: { account: '123456789012', region: 'us-east-1' },
      });
      template = Template.fromStack(stack);
    });

    test('creates DynamoDB table with correct naming', () => {
      template.hasResourceProperties('AWS::DynamoDB::Table', {
        TableName: 'testapp-dev-digistratum',
      });
    });

    test('creates DynamoDB table with PK/SK', () => {
      template.hasResourceProperties('AWS::DynamoDB::Table', {
        KeySchema: [
          { AttributeName: 'PK', KeyType: 'HASH' },
          { AttributeName: 'SK', KeyType: 'RANGE' },
        ],
      });
    });

    test('creates DynamoDB table with GSI1', () => {
      template.hasResourceProperties('AWS::DynamoDB::Table', {
        GlobalSecondaryIndexes: Match.arrayWith([
          Match.objectLike({
            IndexName: 'GSI1',
            KeySchema: [
              { AttributeName: 'GSI1PK', KeyType: 'HASH' },
              { AttributeName: 'GSI1SK', KeyType: 'RANGE' },
            ],
          }),
        ]),
      });
    });

    test('creates DynamoDB table with TTL enabled', () => {
      template.hasResourceProperties('AWS::DynamoDB::Table', {
        TimeToLiveSpecification: {
          AttributeName: 'TTL',
          Enabled: true,
        },
      });
    });

    test('creates DynamoDB table with DESTROY removal policy for dev', () => {
      template.hasResource('AWS::DynamoDB::Table', {
        DeletionPolicy: 'Delete',
        UpdateReplacePolicy: 'Delete',
      });
    });

    test('creates S3 bucket with correct naming pattern', () => {
      template.hasResourceProperties('AWS::S3::Bucket', {
        BucketName: 'testapp-dev-digistratum-assets-123456789012',
      });
    });

    test('creates S3 bucket with public access blocked', () => {
      template.hasResourceProperties('AWS::S3::Bucket', {
        PublicAccessBlockConfiguration: {
          BlockPublicAcls: true,
          BlockPublicPolicy: true,
          IgnorePublicAcls: true,
          RestrictPublicBuckets: true,
        },
      });
    });

    test('creates S3 bucket with S3-managed encryption', () => {
      template.hasResourceProperties('AWS::S3::Bucket', {
        BucketEncryption: {
          ServerSideEncryptionConfiguration: [
            {
              ServerSideEncryptionByDefault: {
                SSEAlgorithm: 'AES256',
              },
            },
          ],
        },
      });
    });

    test('creates S3 bucket with DESTROY removal policy for dev', () => {
      template.hasResource('AWS::S3::Bucket', {
        DeletionPolicy: 'Delete',
        UpdateReplacePolicy: 'Delete',
      });
    });

    test('creates CDK outputs for cross-stack references', () => {
      template.hasOutput('TableName', {
        Value: { Ref: Match.stringLikeRegexp('Table') },
        Export: { Name: 'testapp-dev-digistratum-table-name' },
      });

      template.hasOutput('BucketName', {
        Value: { Ref: Match.stringLikeRegexp('AssetsBucket') },
        Export: { Name: 'testapp-dev-digistratum-bucket-name' },
      });
    });

    test('applies correct tags', () => {
      // Match each tag individually since arrayWith has ordering constraints
      template.hasResourceProperties('AWS::DynamoDB::Table', {
        Tags: Match.arrayWith([
          Match.objectLike({ Key: 'Application', Value: 'testapp' }),
        ]),
      });
      template.hasResourceProperties('AWS::DynamoDB::Table', {
        Tags: Match.arrayWith([
          Match.objectLike({ Key: 'Environment', Value: 'dev' }),
        ]),
      });
      template.hasResourceProperties('AWS::DynamoDB::Table', {
        Tags: Match.arrayWith([
          Match.objectLike({ Key: 'Ecosystem', Value: 'digistratum' }),
        ]),
      });
      template.hasResourceProperties('AWS::DynamoDB::Table', {
        Tags: Match.arrayWith([
          Match.objectLike({ Key: 'ManagedBy', Value: 'CDK' }),
        ]),
      });
    });
  });

  describe('prod environment', () => {
    let prodTemplate: Template;

    beforeAll(() => {
      const app = new cdk.App();
      const prodStack = new DataStack(app, 'TestDataStackProd', {
        appName: 'testapp',
        envName: 'prod',
        ecosystem: 'leapkick',
        env: { account: '123456789012', region: 'us-east-1' },
      });
      prodTemplate = Template.fromStack(prodStack);
    });

    test('creates DynamoDB table with RETAIN removal policy for prod', () => {
      prodTemplate.hasResource('AWS::DynamoDB::Table', {
        DeletionPolicy: 'Retain',
        UpdateReplacePolicy: 'Retain',
      });
    });

    test('creates S3 bucket with RETAIN removal policy for prod', () => {
      prodTemplate.hasResource('AWS::S3::Bucket', {
        DeletionPolicy: 'Retain',
        UpdateReplacePolicy: 'Retain',
      });
    });

    test('creates S3 bucket with versioning enabled for prod', () => {
      prodTemplate.hasResourceProperties('AWS::S3::Bucket', {
        VersioningConfiguration: {
          Status: 'Enabled',
        },
      });
    });

    test('creates resources with correct ecosystem naming', () => {
      prodTemplate.hasResourceProperties('AWS::DynamoDB::Table', {
        TableName: 'testapp-prod-leapkick',
      });
    });
  });

  describe('stack naming', () => {
    test('follows pattern {appName}-data-{envName}-{ecosystem}', () => {
      const app = new cdk.App();
      const namedStack = new DataStack(app, 'bogusapp-data-dev-digistratum', {
        appName: 'bogusapp',
        envName: 'dev',
        ecosystem: 'digistratum',
      });
      
      expect(namedStack.stackName).toBe('bogusapp-data-dev-digistratum');
    });
  });

  describe('public properties', () => {
    let propStack: DataStack;

    beforeAll(() => {
      const app = new cdk.App();
      propStack = new DataStack(app, 'PropTestStack', {
        appName: 'myapp',
        envName: 'dev',
        ecosystem: 'testeco',
        env: { account: '111111111111', region: 'us-west-2' },
      });
    });

    test('exposes table property', () => {
      expect(propStack.table).toBeDefined();
    });

    test('exposes assetsBucket property', () => {
      expect(propStack.assetsBucket).toBeDefined();
    });

    test('exposes tableName property with correct value', () => {
      expect(propStack.tableName).toBe('myapp-dev-testeco');
    });

    test('exposes bucketName property with correct value', () => {
      expect(propStack.bucketName).toBe('myapp-dev-testeco-assets');
    });
  });
});
