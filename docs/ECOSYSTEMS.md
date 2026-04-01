# Ecosystems Configuration

Multi-ecosystem deployment allows a single app to serve multiple domain ecosystems (e.g., digistratum.com, leapkick.com) from one codebase.

## Quick Start

Copy the template and customize:

```bash
cp ecosystems.yaml.example ecosystems.yaml
# Edit ecosystems.yaml with your app details
```

Or create `ecosystems.yaml` in your app root:

```yaml
version: 1

app:
  name: myapp
  displayName: "My Application"

ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: myapp
```

## Schema Reference

### `version` (required)

Schema version for future compatibility. Currently must be `1`.

```yaml
version: 1
```

### `app` (required)

App metadata used for stack naming and resource identification.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | App name (3-30 chars, lowercase alphanumeric with hyphens) |
| `displayName` | string | No | Human-readable name for UI/logging |

**App name rules:**
- Must start with a letter
- Must end with a letter or number
- Only lowercase letters, numbers, and hyphens
- 3-30 characters

```yaml
app:
  name: crm
  displayName: "DS Customer Relations"
```

### `ecosystems` (required)

List of ecosystems this app participates in.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Ecosystem name (`digistratum` or `leapkick`) |
| `enabled` | boolean | Yes | Whether to deploy to this ecosystem |
| `sso_app_id` | string | No | SSO app ID (defaults to `app.name`) |

**Constraints:**
- At least 1 ecosystem required
- Maximum 12 ecosystems (CloudFront limit: 25 alt domains ÷ 2 per ecosystem)
- Disabled ecosystems are excluded from deployment

## Examples

### Single Ecosystem (Backwards Compatible)

For apps that only need DigiStratum:

```yaml
version: 1

app:
  name: projects
  displayName: "DS Projects"

ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: projects
```

**Result:**
- Dev: `projects.dev.digistratum.com`
- Prod: `projects.digistratum.com`
- Single DataStack, single AppStack

### Multi-Ecosystem (DigiStratum + LeapKick)

For apps serving multiple ecosystems:

```yaml
version: 1

app:
  name: marketplace
  displayName: "Marketplace"

ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: marketplace

  - name: leapkick
    enabled: true
    sso_app_id: marketplace
```

**Result:**
- Dev: `marketplace.dev.digistratum.com`, `marketplace.dev.leapkick.com`
- Prod: `marketplace.digistratum.com`, `marketplace.leapkick.com`
- Two DataStacks (data isolation per ecosystem)
- Single AppStack (one CloudFront distribution, all domains)

### Temporarily Disabling an Ecosystem

Set `enabled: false` to exclude an ecosystem without removing config:

```yaml
ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: myapp

  - name: leapkick
    enabled: false  # Not deployed, but config preserved
    sso_app_id: myapp
```

### Different SSO App IDs per Ecosystem

If your app is registered under different names:

```yaml
ecosystems:
  - name: digistratum
    enabled: true
    sso_app_id: crm

  - name: leapkick
    enabled: true
    sso_app_id: lk-crm  # Different registration in LeapKick
```

## CDK Stack Structure

With `ecosystems.yaml` present, CDK creates:

```
├── {app}-data-{env}-{ecosystem}   # Per ecosystem-env data isolation
│   └── DynamoDB, S3, etc.
│
└── {app}-app-{env}                # Single CF distribution
    └── CloudFront, Lambda, API Gateway
        └── All ecosystem domains as aliases
```

Example for `marketplace` with 2 ecosystems:

```
├── marketplace-data-dev-digistratum
├── marketplace-data-dev-leapkick
├── marketplace-app-dev
│
├── marketplace-data-prod-digistratum
├── marketplace-data-prod-leapkick
└── marketplace-app-prod
```

## Legacy Mode (No ecosystems.yaml)

Without `ecosystems.yaml`, apps deploy in legacy single-ecosystem mode:

- Hardcoded to digistratum.com
- Single stack per environment
- No multi-ecosystem support

This maintains backwards compatibility for existing apps.

## Central Registry

Apps reference ecosystems by name only. The central registry (S3) provides:

- Domain names (dev/prod)
- ACM certificate ARNs
- Route53 zone IDs
- SSO URLs

Apps don't need to know infrastructure details—just declare participation.

## IDE Support

For schema validation and autocomplete, add to VS Code settings:

```json
{
  "yaml.schemas": {
    "./infra/schemas/ecosystems.schema.json": "ecosystems.yaml"
  }
}
```

Or add to the file directly:

```yaml
# yaml-language-server: $schema=./infra/schemas/ecosystems.schema.json
version: 1
...
```

## Related

- [Multi-Ecosystem CDK Architecture](./ARCHITECTURE.md)
- [Central Registry](../infra/lib/registry.ts)
