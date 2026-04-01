/**
 * Multi-Ecosystem AppStack Tests
 *
 * Tests for the refactored AppStack that supports multiple ecosystem domains
 * on a single CloudFront distribution.
 *
 * See: Issue #1763
 */
import * as cdk from 'aws-cdk-lib';
import { Template, Match, Capture } from 'aws-cdk-lib/assertions';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import { AppStack, EcosystemAppConfig, MAX_ECOSYSTEMS } from '../lib/app-stack';

// Helper to create mock DynamoDB tables
function createMockTable(scope: cdk.Stack, id: string, tableName: string): dynamodb.Table {
  return new dynamodb.Table(scope, id, {
    tableName,
    partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
  });
}

// Helper to create ecosystem config
function createEcosystemConfig(
  name: string,
  appName: string,
  isProd: boolean
): EcosystemAppConfig {
  const domain = isProd ? `${name}.com` : `dev.${name}.com`;
  return {
    name,
    domain: `${appName}.${domain}`,
    certArn: `arn:aws:acm:us-east-1:123456789012:certificate/${name}-cert`,
    zoneId: `Z${name.toUpperCase()}ZONEID`,
    ssoUrl: `https://account.${domain}`,
    ssoAppId: appName,
  };
}

describe('Multi-Ecosystem AppStack', () => {
  describe('basic setup with two ecosystems', () => {
    let stack: cdk.Stack;
    let appStack: AppStack;
    let template: Template;
    let dataTables: Map<string, dynamodb.ITable>;

    beforeAll(() => {
      const app = new cdk.App();
      
      // Create mock data tables in a separate stack
      const mockDataStack = new cdk.Stack(app, 'MockDataStack', {
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      dataTables = new Map();
      dataTables.set('digistratum', createMockTable(mockDataStack, 'DSTable', 'testapp-prod-digistratum'));
      dataTables.set('leapkick', createMockTable(mockDataStack, 'LKTable', 'testapp-prod-leapkick'));
      
      // Create AppStack with two ecosystems
      appStack = new AppStack(app, 'testapp-app-prod', {
        appName: 'testapp',
        envName: 'prod',
        ecosystems: [
          createEcosystemConfig('digistratum', 'testapp', true),
          createEcosystemConfig('leapkick', 'testapp', true),
        ],
        dataTables,
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      template = Template.fromStack(appStack);
    });

    test('creates single CloudFront distribution', () => {
      template.resourceCountIs('AWS::CloudFront::Distribution', 1);
    });

    test('CloudFront has all ecosystem domains as aliases', () => {
      template.hasResourceProperties('AWS::CloudFront::Distribution', {
        DistributionConfig: {
          Aliases: Match.arrayWith([
            'testapp.digistratum.com',
            'testapp.leapkick.com',
          ]),
        },
      });
    });

    test('creates single Lambda function', () => {
      template.resourceCountIs('AWS::Lambda::Function', 1);
    });

    test('Lambda has ECOSYSTEM_CONFIG environment variable', () => {
      // ECOSYSTEM_CONFIG uses Fn::Join because it references cross-stack table names
      // We verify the structure contains the expected static parts
      template.hasResourceProperties('AWS::Lambda::Function', {
        Environment: {
          Variables: {
            ECOSYSTEM_CONFIG: Match.objectLike({
              'Fn::Join': Match.arrayWith([
                '',
                Match.arrayWith([
                  Match.stringLikeRegexp('"digistratum".*"domain":"testapp.digistratum.com"'),
                ]),
              ]),
            }),
          },
        },
      });
    });

    test('Lambda ECOSYSTEM_CONFIG contains both ecosystems', () => {
      // Check both ecosystems appear in the joined config
      template.hasResourceProperties('AWS::Lambda::Function', {
        Environment: {
          Variables: {
            ECOSYSTEM_CONFIG: Match.objectLike({
              'Fn::Join': Match.arrayWith([
                '',
                Match.arrayWith([
                  Match.stringLikeRegexp('leapkick.*domain.*testapp.leapkick.com'),
                ]),
              ]),
            }),
          },
        },
      });
    });

    test('Lambda ECOSYSTEM_CONFIG includes SSO URLs', () => {
      // Check SSO URLs are included in the config
      template.hasResourceProperties('AWS::Lambda::Function', {
        Environment: {
          Variables: {
            ECOSYSTEM_CONFIG: Match.objectLike({
              'Fn::Join': Match.arrayWith([
                '',
                Match.arrayWith([
                  Match.stringLikeRegexp('ssoUrl.*account.digistratum.com'),
                ]),
              ]),
            }),
          },
        },
      });
    });

    test('creates Route53 record for each ecosystem', () => {
      template.resourceCountIs('AWS::Route53::RecordSet', 2);
      
      // Check records exist for both domains
      template.hasResourceProperties('AWS::Route53::RecordSet', {
        Name: 'testapp.digistratum.com.',
        Type: 'A',
      });
      
      template.hasResourceProperties('AWS::Route53::RecordSet', {
        Name: 'testapp.leapkick.com.',
        Type: 'A',
      });
    });

    test('creates single API Gateway', () => {
      template.resourceCountIs('AWS::ApiGatewayV2::Api', 1);
    });

    test('creates single frontend S3 bucket', () => {
      template.resourceCountIs('AWS::S3::Bucket', 1);
    });

    test('applies correct tags', () => {
      template.hasResourceProperties('AWS::Lambda::Function', {
        Tags: Match.arrayWith([
          Match.objectLike({ Key: 'Application', Value: 'testapp' }),
        ]),
      });
      
      template.hasResourceProperties('AWS::Lambda::Function', {
        Tags: Match.arrayWith([
          Match.objectLike({ Key: 'Environment', Value: 'prod' }),
        ]),
      });
      
      template.hasResourceProperties('AWS::Lambda::Function', {
        Tags: Match.arrayWith([
          Match.objectLike({ Key: 'EcosystemCount', Value: '2' }),
        ]),
      });
    });

    test('outputs all domain URLs', () => {
      template.hasOutput('PrimaryDomainUrl', {
        Value: 'https://testapp.digistratum.com',
      });
      
      template.hasOutput('AllDomains', {
        Value: 'testapp.digistratum.com, testapp.leapkick.com',
      });
    });

    test('exposes public properties', () => {
      expect(appStack.distribution).toBeDefined();
      expect(appStack.api).toBeDefined();
      expect(appStack.apiFunction).toBeDefined();
    });
  });

  describe('ecosystem limit enforcement', () => {
    test('allows up to 12 ecosystems', () => {
      const app = new cdk.App();
      
      // Create 12 mock tables
      const mockDataStack = new cdk.Stack(app, 'MockDataStack12', {
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      const dataTables = new Map<string, dynamodb.ITable>();
      const ecosystems: EcosystemAppConfig[] = [];
      
      for (let i = 1; i <= 12; i++) {
        const name = `eco${i}`;
        dataTables.set(name, createMockTable(mockDataStack, `Table${i}`, `app-prod-${name}`));
        ecosystems.push({
          name,
          domain: `app.${name}.com`,
          certArn: `arn:aws:acm:us-east-1:123456789012:certificate/${name}`,
          zoneId: `Z${name.toUpperCase()}`,
          ssoUrl: `https://account.${name}.com`,
          ssoAppId: 'app',
        });
      }
      
      // Should not throw
      expect(() => {
        new AppStack(app, 'app-app-prod-12', {
          appName: 'app',
          envName: 'prod',
          ecosystems,
          dataTables,
          env: { account: '123456789012', region: 'us-east-1' },
        });
      }).not.toThrow();
    });

    test('throws error for 13+ ecosystems', () => {
      const app = new cdk.App();
      
      // Create 13 mock tables
      const mockDataStack = new cdk.Stack(app, 'MockDataStack13', {
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      const dataTables = new Map<string, dynamodb.ITable>();
      const ecosystems: EcosystemAppConfig[] = [];
      
      for (let i = 1; i <= 13; i++) {
        const name = `eco${i}`;
        dataTables.set(name, createMockTable(mockDataStack, `Table${i}`, `app-prod-${name}`));
        ecosystems.push({
          name,
          domain: `app.${name}.com`,
          certArn: `arn:aws:acm:us-east-1:123456789012:certificate/${name}`,
          zoneId: `Z${name.toUpperCase()}`,
          ssoUrl: `https://account.${name}.com`,
          ssoAppId: 'app',
        });
      }
      
      expect(() => {
        new AppStack(app, 'app-app-prod-13', {
          appName: 'app',
          envName: 'prod',
          ecosystems,
          dataTables,
          env: { account: '123456789012', region: 'us-east-1' },
        });
      }).toThrow(/Too many ecosystems: 13 configured, max is 12/);
    });

    test('throws error for empty ecosystems array', () => {
      const app = new cdk.App();
      
      expect(() => {
        new AppStack(app, 'app-app-prod-empty', {
          appName: 'app',
          envName: 'prod',
          ecosystems: [],
          dataTables: new Map(),
          env: { account: '123456789012', region: 'us-east-1' },
        });
      }).toThrow(/Multi-ecosystem mode requires at least one ecosystem/);
    });

    test('MAX_ECOSYSTEMS constant is exported and equals 12', () => {
      expect(MAX_ECOSYSTEMS).toBe(12);
    });
  });

  describe('missing data table handling', () => {
    test('throws error when data table not provided for ecosystem', () => {
      const app = new cdk.App();
      
      // Create tables for only one ecosystem
      const mockDataStack = new cdk.Stack(app, 'MockDataStackPartial', {
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      const dataTables = new Map<string, dynamodb.ITable>();
      dataTables.set('digistratum', createMockTable(mockDataStack, 'DSTable', 'app-prod-digistratum'));
      // Note: leapkick table is missing
      
      expect(() => {
        new AppStack(app, 'app-app-prod-missing', {
          appName: 'app',
          envName: 'prod',
          ecosystems: [
            createEcosystemConfig('digistratum', 'app', true),
            createEcosystemConfig('leapkick', 'app', true),
          ],
          dataTables,
          env: { account: '123456789012', region: 'us-east-1' },
        });
      }).toThrow(/No data table provided for ecosystem 'leapkick'/);
    });
  });

  describe('dev environment', () => {
    let devTemplate: Template;

    beforeAll(() => {
      const app = new cdk.App();
      
      const mockDataStack = new cdk.Stack(app, 'MockDataStackDev', {
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      const dataTables = new Map<string, dynamodb.ITable>();
      dataTables.set('digistratum', createMockTable(mockDataStack, 'DSTableDev', 'testapp-dev-digistratum'));
      dataTables.set('leapkick', createMockTable(mockDataStack, 'LKTableDev', 'testapp-dev-leapkick'));
      
      const appStack = new AppStack(app, 'testapp-app-dev', {
        appName: 'testapp',
        envName: 'dev',
        ecosystems: [
          createEcosystemConfig('digistratum', 'testapp', false),
          createEcosystemConfig('leapkick', 'testapp', false),
        ],
        dataTables,
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      devTemplate = Template.fromStack(appStack);
    });

    test('S3 bucket has DELETE removal policy for dev', () => {
      devTemplate.hasResource('AWS::S3::Bucket', {
        DeletionPolicy: 'Delete',
        UpdateReplacePolicy: 'Delete',
      });
    });

    test('Lambda has debug log level for dev', () => {
      devTemplate.hasResourceProperties('AWS::Lambda::Function', {
        Environment: {
          Variables: {
            LOG_LEVEL: 'debug',
          },
        },
      });
    });

    test('CloudFront has dev domains', () => {
      devTemplate.hasResourceProperties('AWS::CloudFront::Distribution', {
        DistributionConfig: {
          Aliases: Match.arrayWith([
            'testapp.dev.digistratum.com',
            'testapp.dev.leapkick.com',
          ]),
        },
      });
    });
  });

  describe('prod environment', () => {
    let prodTemplate: Template;

    beforeAll(() => {
      const app = new cdk.App();
      
      const mockDataStack = new cdk.Stack(app, 'MockDataStackProd2', {
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      const dataTables = new Map<string, dynamodb.ITable>();
      dataTables.set('digistratum', createMockTable(mockDataStack, 'DSTableProd2', 'testapp-prod-digistratum'));
      
      const appStack = new AppStack(app, 'testapp-app-prod-2', {
        appName: 'testapp',
        envName: 'prod',
        ecosystems: [
          createEcosystemConfig('digistratum', 'testapp', true),
        ],
        dataTables,
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      prodTemplate = Template.fromStack(appStack);
    });

    test('S3 bucket has RETAIN removal policy for prod', () => {
      prodTemplate.hasResource('AWS::S3::Bucket', {
        DeletionPolicy: 'Retain',
        UpdateReplacePolicy: 'Retain',
      });
    });

    test('Lambda has info log level for prod', () => {
      prodTemplate.hasResourceProperties('AWS::Lambda::Function', {
        Environment: {
          Variables: {
            LOG_LEVEL: 'info',
          },
        },
      });
    });
  });

  describe('stack description', () => {
    test('includes ecosystem count in description', () => {
      const app = new cdk.App();
      
      const mockDataStack = new cdk.Stack(app, 'MockDataStackDesc', {
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      const dataTables = new Map<string, dynamodb.ITable>();
      dataTables.set('digistratum', createMockTable(mockDataStack, 'T1', 't1'));
      dataTables.set('leapkick', createMockTable(mockDataStack, 'T2', 't2'));
      dataTables.set('otherdomain', createMockTable(mockDataStack, 'T3', 't3'));
      
      const appStack = new AppStack(app, 'myapp-app-prod-desc', {
        appName: 'myapp',
        envName: 'prod',
        ecosystems: [
          createEcosystemConfig('digistratum', 'myapp', true),
          createEcosystemConfig('leapkick', 'myapp', true),
          {
            name: 'otherdomain',
            domain: 'myapp.otherdomain.com',
            certArn: 'arn:aws:acm:us-east-1:123456789012:certificate/other',
            zoneId: 'ZOTHER',
            ssoUrl: 'https://account.otherdomain.com',
            ssoAppId: 'myapp',
          },
        ],
        dataTables,
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      const cfnStack = app.synth().getStackByName(appStack.stackName);
      expect(cfnStack.template.Description).toContain('3 ecosystems');
    });
  });

  describe('single-ecosystem backwards compatibility', () => {
    /**
     * When using multi-ecosystem props with only 1 ecosystem,
     * the stack should still work correctly.
     */
    test('works with single ecosystem in multi-ecosystem props', () => {
      const app = new cdk.App();
      
      const mockDataStack = new cdk.Stack(app, 'MockDataStackSingle', {
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      const dataTables = new Map<string, dynamodb.ITable>();
      dataTables.set('digistratum', createMockTable(mockDataStack, 'DSTableSingle', 'app-prod-digistratum'));
      
      const appStack = new AppStack(app, 'app-app-prod-single', {
        appName: 'app',
        envName: 'prod',
        ecosystems: [
          createEcosystemConfig('digistratum', 'app', true),
        ],
        dataTables,
        env: { account: '123456789012', region: 'us-east-1' },
      });
      
      const singleTemplate = Template.fromStack(appStack);
      
      // Should create all resources
      singleTemplate.resourceCountIs('AWS::CloudFront::Distribution', 1);
      singleTemplate.resourceCountIs('AWS::Lambda::Function', 1);
      singleTemplate.resourceCountIs('AWS::Route53::RecordSet', 1);
      
      // Should have single domain
      singleTemplate.hasResourceProperties('AWS::CloudFront::Distribution', {
        DistributionConfig: {
          Aliases: ['app.digistratum.com'],
        },
      });
    });
  });
});

describe('Legacy SingleEcosystem AppStack', () => {
  /**
   * Ensure legacy single-ecosystem mode still works exactly as before
   */
  describe('legacy props still work', () => {
    let app: cdk.App;
    let template: Template;

    beforeAll(() => {
      // Use a fresh app to avoid resource contamination from other tests
      app = new cdk.App();
      const stack = new AppStack(app, 'legacy-app-dev', {
        envName: 'dev',
        domain: 'legacy.dev.digistratum.com',
        certificateArn: 'arn:aws:acm:us-east-1:123456789012:certificate/test',
        env: { account: '123456789012', region: 'us-east-1' },
      });
      template = Template.fromStack(stack);
    });

    test('creates DynamoDB table internally', () => {
      template.hasResourceProperties('AWS::DynamoDB::Table', {
        TableName: 'ds-noc-v2-dev',
      });
    });

    test('creates CloudFront distribution', () => {
      template.resourceCountIs('AWS::CloudFront::Distribution', 1);
    });

    test('creates app Lambda function', () => {
      // Note: There may be additional CustomResource Lambdas created by CDK
      // We verify our app Lambda exists with expected name pattern
      template.hasResourceProperties('AWS::Lambda::Function', {
        FunctionName: Match.stringLikeRegexp('legacy-dev-digistratum-com-api'),
      });
    });

    test('Lambda has DYNAMODB_TABLE env var (not ECOSYSTEM_CONFIG)', () => {
      template.hasResourceProperties('AWS::Lambda::Function', {
        Environment: {
          Variables: {
            DYNAMODB_TABLE: 'ds-noc-v2-dev',
          },
        },
      });
    });
  });
});
