/**
 * DS App Stack Template
 * Full-stack CDK deployment for DigiStratum applications
 *
 * Supports two modes:
 * - Single-ecosystem (backwards compatible): Creates DynamoDB table internally
 * - Multi-ecosystem: Single CF distribution with all ecosystem domains as aliases
 */
import * as cdk from 'aws-cdk-lib';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigateway from 'aws-cdk-lib/aws-apigatewayv2';
import * as apigatewayIntegrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as cloudfront from 'aws-cdk-lib/aws-cloudfront';
import * as origins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as acm from 'aws-cdk-lib/aws-certificatemanager';
import * as route53 from 'aws-cdk-lib/aws-route53';
import * as route53targets from 'aws-cdk-lib/aws-route53-targets';
import * as logs from 'aws-cdk-lib/aws-logs';
import { Construct } from 'constructs';

// -----------------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------------

/**
 * Maximum number of ecosystems supported.
 * CloudFront supports 25 alternate domains; each ecosystem uses 2 (prod + dev).
 * Reserve 1 for primary domain identification.
 */
export const MAX_ECOSYSTEMS = 12;

/**
 * CloudFront Function code for SPA routing.
 *
 * This function rewrites requests to /index.html for SPA client-side routes,
 * while leaving static asset requests unchanged.
 *
 * Why not use errorResponses?
 * - errorResponses apply to ALL origins including API Gateway
 * - When API returns 404/403, CloudFront would intercept and return index.html
 * - This breaks API functionality from the frontend
 *
 * Pattern:
 * - /api/* routes are handled by a separate behavior (not affected)
 * - Requests with file extensions go to S3 unchanged
 * - All other requests (SPA routes) are rewritten to /index.html
 */
const SPA_REWRITE_FUNCTION_CODE = `
function handler(event) {
  var request = event.request;
  var uri = request.uri;

  // If the URI has a file extension, let it through unchanged
  // This handles /assets/*, /images/*, etc.
  if (uri.includes('.')) {
    return request;
  }

  // For SPA routes (no extension), rewrite to index.html
  // Examples: /dashboard, /login, /settings/profile
  request.uri = '/index.html';
  return request;
}
`;

// -----------------------------------------------------------------------------
// Type Definitions
// -----------------------------------------------------------------------------

/**
 * Configuration for a single ecosystem in multi-ecosystem mode
 */
export interface EcosystemAppConfig {
  /** Ecosystem identifier (e.g., 'digistratum', 'leapkick') */
  name: string;

  /** Full domain for this ecosystem (e.g., 'myapp.digistratum.com') */
  domain: string;

  /** ACM certificate ARN for this ecosystem's domain (us-east-1) */
  certArn: string;

  /** Route53 hosted zone ID for DNS record creation */
  zoneId: string;

  /** SSO provider base URL (e.g., 'https://account.digistratum.com') */
  ssoUrl: string;

  /** SSO app ID for this ecosystem */
  ssoAppId: string;
}

/**
 * Props for AppStack in multi-ecosystem mode
 */
export interface MultiEcosystemAppStackProps extends cdk.StackProps {
  /** Application name (e.g., 'bogusapp') */
  appName: string;

  /** Environment name: 'dev' or 'prod' */
  envName: 'dev' | 'prod';

  /** Array of ecosystem configurations (max 12) */
  ecosystems: EcosystemAppConfig[];

  /**
   * Map of ecosystem name to DynamoDB table for data access
   * Created by DataStack for each ecosystem
   */
  dataTables: Map<string, dynamodb.ITable>;
}

/**
 * Legacy props for backwards compatibility (single-ecosystem)
 */
export interface SingleEcosystemAppStackProps extends cdk.StackProps {
  envName: string;
  domain: string;
  certificateArn?: string;
}

/**
 * Combined props type
 */
export type AppStackProps = SingleEcosystemAppStackProps | MultiEcosystemAppStackProps;

/**
 * Type guard to distinguish multi-ecosystem props
 */
function isMultiEcosystem(props: AppStackProps): props is MultiEcosystemAppStackProps {
  return 'ecosystems' in props && Array.isArray(props.ecosystems);
}

// -----------------------------------------------------------------------------
// AppStack Implementation
// -----------------------------------------------------------------------------

export class AppStack extends cdk.Stack {
  /** CloudFront distribution */
  public readonly distribution: cloudfront.Distribution;

  /** API Gateway HTTP API */
  public readonly api: apigateway.HttpApi;

  /** Lambda function */
  public readonly apiFunction: lambda.Function;

  constructor(scope: Construct, id: string, props: AppStackProps) {
    // Validate multi-ecosystem limit before super() call
    if (isMultiEcosystem(props)) {
      if (props.ecosystems.length > MAX_ECOSYSTEMS) {
        throw new Error(
          `Too many ecosystems: ${props.ecosystems.length} configured, max is ${MAX_ECOSYSTEMS}. ` +
            `CloudFront supports 25 alternate domains; each ecosystem uses 2 (prod + dev). ` +
            `Current ecosystems: ${props.ecosystems.map((e) => e.name).join(', ')}`
        );
      }
      if (props.ecosystems.length === 0) {
        throw new Error('Multi-ecosystem mode requires at least one ecosystem');
      }
    }

    super(scope, id, {
      ...props,
      description: isMultiEcosystem(props)
        ? `${props.appName} (${props.envName}) - ${props.ecosystems.length} ecosystems`
        : `ds-noc-v2 (${props.envName}) - https://${props.domain}`,
    });

    if (isMultiEcosystem(props)) {
      this.buildMultiEcosystem(props);
    } else {
      this.buildSingleEcosystem(props);
    }
  }

  /**
   * Multi-ecosystem deployment mode
   * Single CF distribution with all ecosystem domains as aliases
   */
  private buildMultiEcosystem(props: MultiEcosystemAppStackProps): void {
    const { appName, envName, ecosystems, dataTables } = props;
    const isProd = envName === 'prod';

    // Primary ecosystem (first in list) - used for naming and primary domain
    const primaryEcosystem = ecosystems[0];
    const resourcePrefix = `${appName}-${envName}`;

    // Apply standard tags
    cdk.Tags.of(this).add('Application', appName);
    cdk.Tags.of(this).add('Environment', envName);
    cdk.Tags.of(this).add('EcosystemCount', String(ecosystems.length));
    cdk.Tags.of(this).add('ManagedBy', 'CDK');

    // CloudWatch Log Group
    const apiLogGroup = new logs.LogGroup(this, 'ApiLogGroup', {
      logGroupName: `/aws/lambda/${resourcePrefix}-api`,
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Build ECOSYSTEM_CONFIG for Lambda environment
    // This allows the Lambda to route requests by Host header
    const ecosystemConfig: Record<
      string,
      {
        domain: string;
        tableName: string;
        ssoUrl: string;
        ssoAppId: string;
      }
    > = {};

    for (const eco of ecosystems) {
      const table = dataTables.get(eco.name);
      if (!table) {
        throw new Error(
          `No data table provided for ecosystem '${eco.name}'. ` +
            `Ensure DataStack was created and passed to AppStack.`
        );
      }
      ecosystemConfig[eco.name] = {
        domain: eco.domain,
        tableName: table.tableName,
        ssoUrl: eco.ssoUrl,
        ssoAppId: eco.ssoAppId,
      };
    }

    // Lambda Function - single function serving all ecosystems
    (this as { apiFunction: lambda.Function }).apiFunction = new lambda.Function(
      this,
      'ApiFunction',
      {
        functionName: `${resourcePrefix}-api`,
        runtime: lambda.Runtime.PROVIDED_AL2023,
        architecture: lambda.Architecture.ARM_64,
        handler: 'bootstrap',
        code: lambda.Code.fromAsset('../backend/dist'),
        memorySize: 256,
        timeout: cdk.Duration.seconds(30),
        logGroup: apiLogGroup,
        environment: {
          // Multi-ecosystem config (Lambda routes by Host header)
          ECOSYSTEM_CONFIG: JSON.stringify(ecosystemConfig),
          // Default to primary ecosystem for backwards compat
          APP_URL: `https://${primaryEcosystem.domain}`,
          DSACCOUNT_SSO_URL: primaryEcosystem.ssoUrl,
          DSACCOUNT_APP_ID: primaryEcosystem.ssoAppId,
          LOG_LEVEL: isProd ? 'info' : 'debug',
          SERVICE_NAME: appName,
          APP_VERSION: '1.0.0',
        },
      }
    );

    // Grant read/write to all ecosystem tables
    for (const [ecoName, table] of dataTables) {
      table.grantReadWriteData(this.apiFunction);
    }

    // API Gateway
    (this as { api: apigateway.HttpApi }).api = new apigateway.HttpApi(this, 'Api', {
      apiName: `${resourcePrefix}-api`,
    });

    this.api.addRoutes({
      path: '/api/{proxy+}',
      methods: [apigateway.HttpMethod.ANY],
      integration: new apigatewayIntegrations.HttpLambdaIntegration(
        'LambdaIntegration',
        this.apiFunction
      ),
    });

    // S3 Bucket for Frontend (shared across ecosystems)
    const frontendBucket = new s3.Bucket(this, 'FrontendBucket', {
      bucketName: `${resourcePrefix}-frontend-${this.account}`,
      removalPolicy: isProd ? cdk.RemovalPolicy.RETAIN : cdk.RemovalPolicy.DESTROY,
      autoDeleteObjects: !isProd,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
    });

    // Collect all certificates (one per ecosystem)
    // CloudFront allows associating multiple certificates via ViewerCertificate
    // However, with ACM, we need all domains on one cert OR use SNI with multiple certs
    // For simplicity, we'll use the primary ecosystem's certificate
    // and require all domains to be on that cert (wildcard or SAN cert)
    //
    // Future enhancement: Use Certificate Manager to create a single SAN cert
    // covering all ecosystem domains
    const primaryCertificate = acm.Certificate.fromCertificateArn(
      this,
      'PrimaryCertificate',
      primaryEcosystem.certArn
    );

    // Collect all domain names for CloudFront aliases
    const allDomains = ecosystems.map((e) => e.domain);

    // CloudFront Origin Access Control
    const oac = new cloudfront.S3OriginAccessControl(this, 'OAC', {
      signing: cloudfront.Signing.SIGV4_NO_OVERRIDE,
    });

    // CloudFront Function for SPA routing
    // Rewrites requests without file extensions to /index.html
    // This avoids using errorResponses which would also intercept API 404/403s
    const spaRewriteFunction = new cloudfront.Function(this, 'SpaRewriteFunction', {
      functionName: `${resourcePrefix}-spa-rewrite`,
      code: cloudfront.FunctionCode.fromInline(SPA_REWRITE_FUNCTION_CODE),
      runtime: cloudfront.FunctionRuntime.JS_2_0,
      comment: 'Rewrite SPA routes to index.html while preserving API and asset paths',
    });

    // CloudFront Distribution - single distribution, all domains
    (this as { distribution: cloudfront.Distribution }).distribution =
      new cloudfront.Distribution(this, 'Distribution', {
        defaultBehavior: {
          origin: origins.S3BucketOrigin.withOriginAccessControl(frontendBucket, {
            originAccessControl: oac,
          }),
          viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
          cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
          functionAssociations: [
            {
              function: spaRewriteFunction,
              eventType: cloudfront.FunctionEventType.VIEWER_REQUEST,
            },
          ],
        },
        additionalBehaviors: {
          '/api/*': {
            origin: new origins.HttpOrigin(
              `${this.api.httpApiId}.execute-api.${this.region}.amazonaws.com`
            ),
            viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
            allowedMethods: cloudfront.AllowedMethods.ALLOW_ALL,
            cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
            originRequestPolicy: cloudfront.OriginRequestPolicy.ALL_VIEWER_EXCEPT_HOST_HEADER,
          },
        },
        defaultRootObject: 'index.html',
        // NOTE: No errorResponses here - they would intercept API 404/403s too.
        // SPA fallback is handled by the SpaRewriteFunction above.
        // All ecosystem domains as aliases
        domainNames: allDomains,
        certificate: primaryCertificate,
      });

    // Route53 DNS records for each ecosystem
    // Cache hosted zones to avoid duplicate lookups
    const hostedZoneCache = new Map<string, route53.IHostedZone>();

    for (const eco of ecosystems) {
      // Look up or reuse hosted zone
      let hostedZone = hostedZoneCache.get(eco.zoneId);
      if (!hostedZone) {
        hostedZone = route53.HostedZone.fromHostedZoneAttributes(
          this,
          `HostedZone-${eco.name}`,
          {
            hostedZoneId: eco.zoneId,
            // Zone name derived from domain (app.domain.com -> domain.com)
            zoneName: eco.domain.split('.').slice(1).join('.'),
          }
        );
        hostedZoneCache.set(eco.zoneId, hostedZone);
      }

      // Create A record alias to CloudFront
      new route53.ARecord(this, `AliasRecord-${eco.name}`, {
        zone: hostedZone,
        recordName: eco.domain,
        target: route53.RecordTarget.fromAlias(
          new route53targets.CloudFrontTarget(this.distribution)
        ),
      });
    }

    // Outputs
    new cdk.CfnOutput(this, 'PrimaryDomainUrl', {
      value: `https://${primaryEcosystem.domain}`,
      description: 'Primary application URL',
    });

    new cdk.CfnOutput(this, 'AllDomains', {
      value: allDomains.join(', '),
      description: 'All ecosystem domains',
    });

    new cdk.CfnOutput(this, 'CloudFrontUrl', {
      value: `https://${this.distribution.distributionDomainName}`,
      description: 'CloudFront URL (fallback)',
    });

    new cdk.CfnOutput(this, 'ApiUrl', {
      value: this.api.url!,
      description: 'API Gateway URL (direct)',
    });

    new cdk.CfnOutput(this, 'FrontendBucketName', {
      value: frontendBucket.bucketName,
      description: 'S3 bucket for frontend deployment',
    });

    new cdk.CfnOutput(this, 'CloudFrontDistributionId', {
      value: this.distribution.distributionId,
      description: 'CloudFront distribution ID for cache invalidation',
    });

    new cdk.CfnOutput(this, 'EcosystemCount', {
      value: String(ecosystems.length),
      description: 'Number of ecosystems configured',
    });
  }

  /**
   * Single-ecosystem deployment mode (backwards compatible)
   * Creates DynamoDB table internally
   */
  private buildSingleEcosystem(props: SingleEcosystemAppStackProps): void {
    const { envName, domain } = props;
    const isProd = envName === 'prod';

    // Resource naming: replace dots with dashes for AWS resources that don't allow dots
    const resourcePrefix = domain.replace(/\./g, '-');
    const tableName = `ds-noc-v2-${envName}`;

    // Apply standard tags
    cdk.Tags.of(this).add('Application', 'ds-noc-v2');
    cdk.Tags.of(this).add('Environment', envName);
    cdk.Tags.of(this).add('Domain', domain);
    cdk.Tags.of(this).add('ManagedBy', 'CDK');

    // CloudWatch Log Group
    const apiLogGroup = new logs.LogGroup(this, 'ApiLogGroup', {
      logGroupName: `/aws/lambda/${resourcePrefix}-api`,
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // DynamoDB Table
    const table = new dynamodb.Table(this, 'Table', {
      tableName,
      partitionKey: { name: 'PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'SK', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: isProd ? cdk.RemovalPolicy.RETAIN : cdk.RemovalPolicy.DESTROY,
      timeToLiveAttribute: 'TTL',
    });

    // Add GSI for common query patterns
    table.addGlobalSecondaryIndex({
      indexName: 'GSI1',
      partitionKey: { name: 'GSI1PK', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'GSI1SK', type: dynamodb.AttributeType.STRING },
    });

    // Lambda Function
    (this as { apiFunction: lambda.Function }).apiFunction = new lambda.Function(
      this,
      'ApiFunction',
      {
        functionName: `${resourcePrefix}-api`,
        runtime: lambda.Runtime.PROVIDED_AL2023,
        architecture: lambda.Architecture.ARM_64,
        handler: 'bootstrap',
        code: lambda.Code.fromAsset('../backend/dist'),
        memorySize: 256,
        timeout: cdk.Duration.seconds(30),
        logGroup: apiLogGroup,
        environment: {
          DYNAMODB_TABLE: tableName,
          APP_URL: `https://${domain}`,
          DSACCOUNT_SSO_URL: isProd
            ? 'https://account.digistratum.com'
            : 'https://account.dev.digistratum.com',
          DSACCOUNT_APP_ID: `dsnocv2`,
          LOG_LEVEL: isProd ? 'info' : 'debug',
          SERVICE_NAME: 'ds-noc-v2',
          APP_VERSION: '1.0.0',
        },
      }
    );
    table.grantReadWriteData(this.apiFunction);

    // API Gateway
    (this as { api: apigateway.HttpApi }).api = new apigateway.HttpApi(this, 'Api', {
      apiName: `${resourcePrefix}-api`,
    });

    this.api.addRoutes({
      path: '/api/{proxy+}',
      methods: [apigateway.HttpMethod.ANY],
      integration: new apigatewayIntegrations.HttpLambdaIntegration(
        'LambdaIntegration',
        this.apiFunction
      ),
    });

    // S3 Bucket for Frontend
    const frontendBucket = new s3.Bucket(this, 'FrontendBucket', {
      bucketName: `${resourcePrefix}-frontend-${this.account}`,
      removalPolicy: isProd ? cdk.RemovalPolicy.RETAIN : cdk.RemovalPolicy.DESTROY,
      autoDeleteObjects: !isProd,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
    });

    // Certificate
    let certificate: acm.ICertificate | undefined;
    if (props.certificateArn) {
      certificate = acm.Certificate.fromCertificateArn(
        this,
        'Certificate',
        props.certificateArn
      );
    }

    // CloudFront Origin Access Control
    const oac = new cloudfront.S3OriginAccessControl(this, 'OAC', {
      signing: cloudfront.Signing.SIGV4_NO_OVERRIDE,
    });

    // CloudFront Function for SPA routing
    // Rewrites requests without file extensions to /index.html
    // This avoids using errorResponses which would also intercept API 404/403s
    const spaRewriteFunction = new cloudfront.Function(this, 'SpaRewriteFunction', {
      functionName: `${resourcePrefix}-spa-rewrite`,
      code: cloudfront.FunctionCode.fromInline(SPA_REWRITE_FUNCTION_CODE),
      runtime: cloudfront.FunctionRuntime.JS_2_0,
      comment: 'Rewrite SPA routes to index.html while preserving API and asset paths',
    });

    // CloudFront Distribution
    (this as { distribution: cloudfront.Distribution }).distribution =
      new cloudfront.Distribution(this, 'Distribution', {
        defaultBehavior: {
          origin: origins.S3BucketOrigin.withOriginAccessControl(frontendBucket, {
            originAccessControl: oac,
          }),
          viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
          cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
          functionAssociations: [
            {
              function: spaRewriteFunction,
              eventType: cloudfront.FunctionEventType.VIEWER_REQUEST,
            },
          ],
        },
        additionalBehaviors: {
          '/api/*': {
            origin: new origins.HttpOrigin(
              `${this.api.httpApiId}.execute-api.${this.region}.amazonaws.com`
            ),
            viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
            allowedMethods: cloudfront.AllowedMethods.ALLOW_ALL,
            cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
            originRequestPolicy: cloudfront.OriginRequestPolicy.ALL_VIEWER_EXCEPT_HOST_HEADER,
          },
        },
        defaultRootObject: 'index.html',
        // NOTE: No errorResponses here - they would intercept API 404/403s too.
        // SPA fallback is handled by the SpaRewriteFunction above.
        domainNames: certificate ? [domain] : undefined,
        certificate,
      });

    // Route53 DNS (if certificate is provided, we have DNS)
    if (certificate) {
      // Determine hosted zone from domain
      const zoneParts = domain.split('.');
      // For dev.digistratum.com subdomains, use dev.digistratum.com zone
      // For digistratum.com subdomains, use digistratum.com zone
      const zoneName = zoneParts.slice(-3).join('.').includes('dev.')
        ? zoneParts.slice(-3).join('.')
        : zoneParts.slice(-2).join('.');

      const hostedZone = route53.HostedZone.fromLookup(this, 'HostedZone', {
        domainName: zoneName,
      });

      new route53.ARecord(this, 'AliasRecord', {
        zone: hostedZone,
        recordName: domain,
        target: route53.RecordTarget.fromAlias(
          new route53targets.CloudFrontTarget(this.distribution)
        ),
      });
    }

    // Outputs
    new cdk.CfnOutput(this, 'DomainUrl', {
      value: `https://${domain}`,
      description: 'Application URL',
    });

    new cdk.CfnOutput(this, 'CloudFrontUrl', {
      value: `https://${this.distribution.distributionDomainName}`,
      description: 'CloudFront URL (fallback)',
    });

    new cdk.CfnOutput(this, 'ApiUrl', {
      value: this.api.url!,
      description: 'API Gateway URL (direct)',
    });

    new cdk.CfnOutput(this, 'FrontendBucketName', {
      value: frontendBucket.bucketName,
      description: 'S3 bucket for frontend deployment',
    });

    new cdk.CfnOutput(this, 'CloudFrontDistributionId', {
      value: this.distribution.distributionId,
      description: 'CloudFront distribution ID for cache invalidation',
    });
  }
}
