#!/usr/bin/env node
/**
 * DS App Template CDK Entry Point
 *
 * Orchestrates multi-ecosystem deployment:
 * 1. Load ecosystems.yaml from app root
 * 2. Fetch central registry for ecosystem metadata
 * 3. Merge app config with registry data
 * 4. Create DataStack per ecosystem-env combination
 * 5. Create single AppStack per env with all ecosystem configs
 *
 * Stack naming:
 *   {app}-data-{env}-{ecosystem}  # Per ecosystem-env data isolation
 *   {app}-app-{env}               # Single CF distribution for all ecosystems
 */
import * as cdk from 'aws-cdk-lib';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as fs from 'fs';
import * as path from 'path';
import * as YAML from 'yaml';
import { AppStack } from '../lib/app-stack';
import { DataStack } from '../lib/data-stack';
import { fetchEcosystemRegistry, EcosystemRegistryClient } from '../lib/registry';

// -----------------------------------------------------------------------------
// Type Definitions
// -----------------------------------------------------------------------------

/** App-level ecosystem configuration (from ecosystems.yaml) */
interface AppEcosystemEntry {
  name: string;
  enabled: boolean;
  sso_app_id?: string;
}

/** App ecosystems.yaml schema */
interface AppEcosystemsConfig {
  version: number;
  app: {
    name: string;
    displayName?: string;
  };
  ecosystems: AppEcosystemEntry[];
}

/** Merged ecosystem config for stack creation */
interface ResolvedEcosystemConfig {
  name: string;
  domain: string;
  certArn: string;
  zoneId: string;
  ssoUrl: string;
  ssoAppId: string;
}

// -----------------------------------------------------------------------------
// Configuration Loading
// -----------------------------------------------------------------------------

const MAX_ECOSYSTEMS = 12;

/**
 * Load and parse ecosystems.yaml from app root
 */
function loadAppEcosystems(appRoot: string): AppEcosystemsConfig | null {
  const configPath = path.join(appRoot, 'ecosystems.yaml');

  if (!fs.existsSync(configPath)) {
    // No ecosystems.yaml = single-ecosystem legacy mode
    return null;
  }

  const content = fs.readFileSync(configPath, 'utf-8');
  const config = YAML.parse(content) as AppEcosystemsConfig;

  // Validate schema version
  if (!config.version || config.version < 1) {
    throw new Error(`Invalid ecosystems.yaml: missing or invalid version field`);
  }

  // Validate app section
  if (!config.app?.name) {
    throw new Error(`Invalid ecosystems.yaml: missing app.name`);
  }

  // Validate ecosystems array
  if (!Array.isArray(config.ecosystems) || config.ecosystems.length === 0) {
    throw new Error(`Invalid ecosystems.yaml: ecosystems must be a non-empty array`);
  }

  // Validate ecosystem entries
  for (const eco of config.ecosystems) {
    if (!eco.name) {
      throw new Error(`Invalid ecosystems.yaml: ecosystem entry missing name`);
    }
    if (typeof eco.enabled !== 'boolean') {
      throw new Error(`Invalid ecosystems.yaml: ecosystem '${eco.name}' missing enabled flag`);
    }
  }

  // Filter to enabled ecosystems
  const enabledEcosystems = config.ecosystems.filter(e => e.enabled);

  // Enforce hard limit
  if (enabledEcosystems.length > MAX_ECOSYSTEMS) {
    throw new Error(
      `Too many ecosystems: ${enabledEcosystems.length} enabled, max is ${MAX_ECOSYSTEMS}. ` +
        `CloudFront supports 25 alternate domains; each ecosystem uses 2 (prod + dev).`
    );
  }

  return {
    ...config,
    ecosystems: enabledEcosystems,
  };
}

/**
 * Resolve ecosystem configs by merging app config with central registry
 */
function resolveEcosystems(
  appConfig: AppEcosystemsConfig,
  registry: EcosystemRegistryClient,
  envName: 'dev' | 'prod'
): ResolvedEcosystemConfig[] {
  return appConfig.ecosystems.map(appEco => {
    // Ensure ecosystem exists in registry
    if (!registry.hasEcosystem(appEco.name)) {
      throw new Error(
        `Ecosystem '${appEco.name}' in ecosystems.yaml not found in central registry. ` +
          `Available ecosystems: ${registry.getEcosystemNames().join(', ')}`
      );
    }

    const registryEco = registry.getEcosystem(appEco.name);
    const domain = registry.getDomain(appEco.name, envName);

    return {
      name: appEco.name,
      domain: `${appConfig.app.name}.${domain}`,
      certArn: registry.getCertArn(appEco.name, envName),
      zoneId: registry.getZoneId(appEco.name, envName),
      ssoUrl: registry.getSsoUrl(appEco.name, envName),
      ssoAppId: appEco.sso_app_id || appConfig.app.name,
    };
  });
}

// -----------------------------------------------------------------------------
// Legacy Single-Ecosystem Config
// -----------------------------------------------------------------------------

/** Legacy hardcoded config for apps without ecosystems.yaml */
const LEGACY_CONFIG: Record<string, { domain: string; certArn?: string }> = {
  dev: {
    domain: 'noc-v2.dev.digistratum.com',
    certArn: 'arn:aws:acm:us-east-1:171949636152:certificate/2129713f-4dc8-4248-98ee-a6a8df16dff4',
  },
  prod: {
    domain: 'noc-v2.digistratum.com',
    certArn: 'arn:aws:acm:us-east-1:171949636152:certificate/723daeca-019a-43de-b03d-0bcdf37c2768',
  },
};

// -----------------------------------------------------------------------------
// Main CDK App
// -----------------------------------------------------------------------------

const app = new cdk.App();

// Get environment from CDK context
const envName = (app.node.tryGetContext('env') || 'dev') as 'dev' | 'prod';

// Get stacks filter from CDK context
// Passed via `-c stacks=pattern` to avoid constructing unnecessary stacks
// This prevents asset resolution errors when deploying data stacks only
const stacksFilter = app.node.tryGetContext('stacks') as string | undefined;

/**
 * Check if a stack should be created based on the --context stacks=pattern filter.
 * If no filter is provided, all stacks are created.
 * Supports glob-like patterns with * wildcards.
 * 
 * Usage: cdk deploy -c stacks='myapp-data-*' 'myapp-data-*'
 * The context parameter prevents stack construction, the positional arg filters deployment.
 */
function shouldCreateStack(stackId: string): boolean {
  if (!stacksFilter) {
    return true; // No filter = create all stacks
  }
  
  // Convert glob pattern to regex
  // e.g., 'myapp-data-dev-*' -> /^myapp-data-dev-.*$/
  const patterns = stacksFilter.split(',').map(p => p.trim());
  
  for (const pattern of patterns) {
    // Escape regex special chars except *, then convert * to .*
    const regexPattern = pattern
      .replace(/[.+?^${}()|[\]\\]/g, '\\$&')
      .replace(/\*/g, '.*');
    const regex = new RegExp(`^${regexPattern}$`);
    
    if (regex.test(stackId)) {
      return true;
    }
  }
  
  return false;
}

// AWS environment configuration
const awsEnv = {
  account: process.env.CDK_DEFAULT_ACCOUNT || '171949636152',
  region: process.env.CDK_DEFAULT_REGION || 'us-east-1',
};

// Find app root (parent of infra/)
const infraDir = process.cwd();
const appRoot = path.resolve(infraDir, '..');

// Attempt to load ecosystems.yaml
const appConfig = loadAppEcosystems(appRoot);

if (appConfig) {
  // -------------------------------------------------------------------------
  // Ecosystems Mode (from ecosystems.yaml)
  // -------------------------------------------------------------------------
  // Always use consistent stack naming pattern regardless of ecosystem count:
  //   {app}-data-{env}-{ecosystem}  # DataStack per ecosystem
  //   {app}-app-{env}               # Single AppStack
  //
  // This ensures CI/CD workflows have predictable stack names.
  console.log(`Ecosystems mode: ${appConfig.ecosystems.length} ecosystem(s)`);
  
  // Fetch central registry
  const registry = fetchEcosystemRegistry();
  
  // Resolve ecosystem configurations
  const resolvedEcosystems = resolveEcosystems(appConfig, registry, envName);
  
  // Track data stacks for dependency management
  const dataStacks: DataStack[] = [];
  const dataTables: Map<string, dynamodb.ITable> = new Map();
  
  // Determine which stacks need to be created based on -c stacks=pattern
  const appStackId = `${appConfig.app.name}-app-${envName}`;
  const needsAppStack = shouldCreateStack(appStackId);
  
  // Create DataStack for each ecosystem (if needed)
  for (const eco of resolvedEcosystems) {
    const stackId = `${appConfig.app.name}-data-${envName}-${eco.name}`;
    
    // Only create DataStack if it matches the filter OR if AppStack needs it
    if (shouldCreateStack(stackId) || needsAppStack) {
      const dataStack = new DataStack(app, stackId, {
        appName: appConfig.app.name,
        envName,
        ecosystem: eco.name,
        env: awsEnv,
      });
      
      dataStacks.push(dataStack);
      dataTables.set(eco.name, dataStack.table);
    }
  }
  
  // Create AppStack only if it matches the filter
  // This avoids asset resolution errors when deploying data stacks only
  if (needsAppStack) {
    // Build EcosystemAppConfig array for AppStack
    const ecosystemConfigs = resolvedEcosystems.map(eco => ({
      name: eco.name,
      domain: eco.domain,
      certArn: eco.certArn,
      zoneId: eco.zoneId,
      ssoUrl: eco.ssoUrl,
      ssoAppId: eco.ssoAppId,
    }));
    
    const appStack = new AppStack(app, appStackId, {
      appName: appConfig.app.name,
      envName,
      ecosystems: ecosystemConfigs,
      dataTables,
      env: awsEnv,
    });
    
    // Add explicit dependencies: AppStack depends on all DataStacks
    // CDK handles this via cross-stack references, but we make it explicit
    for (const dataStack of dataStacks) {
      appStack.addDependency(dataStack);
    }
  }
} else {
  // -------------------------------------------------------------------------
  // Legacy Single-Ecosystem Mode (backwards compatible)
  // -------------------------------------------------------------------------
  console.log('Legacy single-ecosystem mode (no ecosystems.yaml found)');
  
  const envConfig = LEGACY_CONFIG[envName];
  if (!envConfig) {
    throw new Error(`Unknown environment: ${envName}`);
  }
  
  new AppStack(app, `ds-noc-v2-${envName}`, {
    envName,
    domain: envConfig.domain,
    certificateArn: envConfig.certArn,
    env: awsEnv,
  });
}
