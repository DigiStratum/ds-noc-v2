/**
 * Ecosystem Registry Utility
 *
 * Fetches ecosystem configuration from the central registry (S3) at CDK synth time.
 * The registry contains shared infrastructure metadata (certs, zone IDs, SSO URLs)
 * that apps reference without hardcoding.
 *
 * Usage:
 *   const ecosystems = await fetchEcosystemRegistry();
 *   const ds = ecosystems.getEcosystem('digistratum');
 *   console.log(ds.prod_cert_arn);
 */

import { execSync } from 'child_process';
import * as YAML from 'yaml';

// -----------------------------------------------------------------------------
// Type Definitions
// -----------------------------------------------------------------------------

/**
 * Single ecosystem configuration
 */
export interface EcosystemConfig {
  /** Ecosystem identifier (e.g., 'digistratum', 'leapkick') */
  name: string;

  /** Production domain (e.g., 'digistratum.com') */
  domain: string;

  /** Development domain (e.g., 'dev.digistratum.com') */
  dev_domain: string;

  /** ACM certificate ARN for production (wildcard, us-east-1) */
  prod_cert_arn: string;

  /** ACM certificate ARN for dev (wildcard, us-east-1) */
  dev_cert_arn: string;

  /** Route53 hosted zone ID for production domain */
  route53_zone_id: string;

  /** Route53 hosted zone ID for dev domain */
  dev_route53_zone_id: string;

  /** SSO provider base URL for production */
  sso_base_url: string;

  /** SSO provider base URL for dev */
  dev_sso_base_url: string;

  /** Pattern for SSO app IDs (e.g., '{appname}.digistratum.com') */
  sso_app_id_pattern?: string;

  /** Human-readable description */
  description?: string;

  /** Contact/owner for this ecosystem */
  contact?: string;

  /** Arbitrary tags */
  tags?: Record<string, string>;
}

/**
 * Registry file structure
 */
export interface EcosystemRegistry {
  /** Schema version */
  version: string;

  /** Map of ecosystem name -> config */
  ecosystems: Record<string, EcosystemConfig>;
}

/**
 * Wrapper class for type-safe registry access
 */
export class EcosystemRegistryClient {
  constructor(private readonly registry: EcosystemRegistry) {}

  /**
   * Get a specific ecosystem by name.
   * Throws if ecosystem not found.
   */
  getEcosystem(name: string): EcosystemConfig {
    const ecosystem = this.registry.ecosystems[name];
    if (!ecosystem) {
      const available = Object.keys(this.registry.ecosystems).join(', ');
      throw new Error(
        `Ecosystem '${name}' not found in registry. Available: ${available}`
      );
    }
    return ecosystem;
  }

  /**
   * Check if an ecosystem exists
   */
  hasEcosystem(name: string): boolean {
    return name in this.registry.ecosystems;
  }

  /**
   * Get all ecosystem names
   */
  getEcosystemNames(): string[] {
    return Object.keys(this.registry.ecosystems);
  }

  /**
   * Get cert ARN for an ecosystem + environment
   */
  getCertArn(ecosystemName: string, env: 'prod' | 'dev'): string {
    const eco = this.getEcosystem(ecosystemName);
    return env === 'prod' ? eco.prod_cert_arn : eco.dev_cert_arn;
  }

  /**
   * Get Route53 zone ID for an ecosystem + environment
   */
  getZoneId(ecosystemName: string, env: 'prod' | 'dev'): string {
    const eco = this.getEcosystem(ecosystemName);
    return env === 'prod' ? eco.route53_zone_id : eco.dev_route53_zone_id;
  }

  /**
   * Get SSO base URL for an ecosystem + environment
   */
  getSsoUrl(ecosystemName: string, env: 'prod' | 'dev'): string {
    const eco = this.getEcosystem(ecosystemName);
    return env === 'prod' ? eco.sso_base_url : eco.dev_sso_base_url;
  }

  /**
   * Get domain for an ecosystem + environment
   */
  getDomain(ecosystemName: string, env: 'prod' | 'dev'): string {
    const eco = this.getEcosystem(ecosystemName);
    return env === 'prod' ? eco.domain : eco.dev_domain;
  }

  /**
   * Get the raw registry data
   */
  getRawRegistry(): EcosystemRegistry {
    return this.registry;
  }
}

// -----------------------------------------------------------------------------
// Registry Fetch
// -----------------------------------------------------------------------------

const REGISTRY_BUCKET = 'ds-infra-config';
const REGISTRY_KEY = 'ecosystems.yaml';

/**
 * Fetch the ecosystem registry from S3.
 *
 * This runs synchronously using aws CLI (required for CDK synth context).
 * The registry is cached in the process for subsequent calls.
 *
 * @throws Error if S3 fetch fails or registry is invalid
 */
let cachedRegistry: EcosystemRegistryClient | null = null;

export function fetchEcosystemRegistry(): EcosystemRegistryClient {
  if (cachedRegistry) {
    return cachedRegistry;
  }

  const s3Uri = `s3://${REGISTRY_BUCKET}/${REGISTRY_KEY}`;

  let yamlContent: string;
  try {
    yamlContent = execSync(`aws s3 cp ${s3Uri} -`, {
      encoding: 'utf-8',
      stdio: ['pipe', 'pipe', 'pipe'],
    });
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    throw new Error(
      `Failed to fetch ecosystem registry from ${s3Uri}: ${message}\n` +
        `Ensure AWS credentials are configured and the registry exists.`
    );
  }

  let registry: EcosystemRegistry;
  try {
    registry = YAML.parse(yamlContent) as EcosystemRegistry;
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    throw new Error(`Failed to parse ecosystem registry YAML: ${message}`);
  }

  // Validate structure
  if (!registry.version) {
    throw new Error('Ecosystem registry missing "version" field');
  }
  if (!registry.ecosystems || typeof registry.ecosystems !== 'object') {
    throw new Error('Ecosystem registry missing "ecosystems" field');
  }

  // Validate each ecosystem has required fields
  const requiredFields: (keyof EcosystemConfig)[] = [
    'name',
    'domain',
    'dev_domain',
    'prod_cert_arn',
    'dev_cert_arn',
    'route53_zone_id',
    'dev_route53_zone_id',
    'sso_base_url',
    'dev_sso_base_url',
  ];

  for (const [name, config] of Object.entries(registry.ecosystems)) {
    for (const field of requiredFields) {
      if (!config[field]) {
        throw new Error(
          `Ecosystem '${name}' missing required field: ${field}`
        );
      }
    }
  }

  cachedRegistry = new EcosystemRegistryClient(registry);
  return cachedRegistry;
}

/**
 * Clear the cached registry (useful for testing)
 */
export function clearRegistryCache(): void {
  cachedRegistry = null;
}
