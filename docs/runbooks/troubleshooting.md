# Troubleshooting Guide

Common issues and solutions when working with DS Ecosystem apps.

## Create-App Issues

### "App name must start with a letter"

**Cause:** App names must be lowercase, start with a letter, and contain only letters, numbers, and hyphens.

**Fix:** Use a valid name:
```bash
./tools/cmd/create-app.sh -n my-app      # ✓ Valid
./tools/cmd/create-app.sh -n 123app      # ✗ Invalid (starts with number)
./tools/cmd/create-app.sh -n My_App      # ✗ Invalid (uppercase, underscore)
```

### "AWS account ID must be 12 digits"

**Cause:** AWS account ID validation failed.

**Fix:** Find your account ID:
```bash
aws sts get-caller-identity --query Account --output text
```

### "Failed to create GitHub repository"

**Possible causes:**
1. Not authenticated with GitHub CLI
2. Don't have org write permissions
3. Repository already exists

**Fix:**
```bash
# Check auth status
gh auth status

# Re-authenticate if needed
gh auth login

# Check if repo exists
gh repo view DigiStratum/my-app
```

### Placeholder tokens not replaced

**Cause:** Script interrupted or token format issues.

**Fix:** Search for remaining placeholders:
```bash
grep -r "__" ~/repos/digistratum/my-app --include="*.ts" --include="*.go" --include="*.json"
```

## Update-Template Issues

### "Not a DS Ecosystem app: missing .template-version"

**Cause:** Target directory is not a valid derived app.

**Fix:** Ensure you're targeting the correct directory:
```bash
ls ~/repos/digistratum/my-app/.template-version
```

### "Already at latest version"

**Cause:** App is already at the same version as the template.

**Fix:** The new script always syncs based on the manifest, so just run it:
```bash
cd ~/repos/digistratum/my-app
./tools/cmd/update-from-template.sh
```

### Template changes lost after update

**Cause:** You modified `template/` files directly.

**Fix:** Template modifications don't survive updates by design:
1. Identify your changes
2. Move them to `app/` layer
3. Re-run update

### pnpm install fails after update

**Cause:** Lock file mismatch or dependency conflicts.

**Fix:**
```bash
rm -rf node_modules pnpm-lock.yaml
pnpm install
```

## Build Issues

### TypeScript errors after update

**Cause:** Breaking changes in template or packages.

**Fix:**
1. Check CHANGELOG for breaking changes
2. Update imports if paths changed
3. Fix type errors based on new interfaces

### Go module errors

**Cause:** Module cache stale or conflicting versions.

**Fix:**
```bash
cd backend
go clean -modcache
go mod tidy
go mod download
```

### Vite build fails

**Cause:** Various configuration or dependency issues.

**Fix:**
```bash
# Clear caches
rm -rf node_modules/.vite frontend/dist

# Reinstall and build
pnpm install
pnpm build
```

## Deployment Issues

### OIDC authentication fails

**Error:** "Could not assume role with OIDC"

**Possible causes:**
1. IAM role trust policy doesn't include repo
2. OIDC provider not configured
3. Wrong role ARN in secrets

**Fix:**
Check trust policy includes your repo:
```json
{
  "Condition": {
    "StringLike": {
      "token.actions.githubusercontent.com:sub": "repo:DigiStratum/my-app:*"
    }
  }
}
```

### CDK deployment fails

**Error:** "Stack is in ROLLBACK_COMPLETE state"

**Fix:** Delete the failed stack:
```bash
aws cloudformation delete-stack --stack-name MyApp-prod
# Wait for deletion, then retry deployment
```

### CloudFront invalidation fails

**Error:** "Distribution not found"

**Cause:** First deployment hasn't completed.

**Fix:** Wait for initial deployment to complete, then retry.

### Lambda cold starts too slow

**Cause:** Large bundle or initialization issues.

**Fix:**
1. Check bundle size: `ls -la backend/dist/`
2. Verify arm64 compilation
3. Consider provisioned concurrency for critical paths

## Local Development Issues

### Port already in use

**Error:** "Port 3000 already in use"

**Fix:**
```bash
# Find and kill process
lsof -i :3000
kill -9 <PID>

# Or use different port
PORT=3001 pnpm dev
```

### Backend can't connect to DynamoDB

**Cause:** Local DynamoDB not running or wrong endpoint.

**Fix:**
```bash
# Start local DynamoDB
docker run -p 8000:8000 amazon/dynamodb-local

# Set environment
export DYNAMODB_ENDPOINT=http://localhost:8000
```

### CORS errors in browser

**Cause:** Backend CORS configuration doesn't allow frontend origin.

**Fix:** Check `template/middleware/cors.go` allows your frontend URL.

### Auth redirects to wrong URL

**Cause:** `APP_URL` environment variable incorrect.

**Fix:**
```bash
export APP_URL=http://localhost:3000
```

## Package Issues

### @digistratum/* package not found

**Cause:** Package URL incorrect or not published.

**Fix:**
1. Check package exists: `curl -I https://packages.digistratum.com/@digistratum/layout/layout-X.X.X.tgz`
2. Verify `.npmrc` has correct registry configuration
3. Update to latest package version

### Version mismatch between packages

**Cause:** Incompatible versions of @digistratum packages.

**Fix:** Update all packages together:
```bash
# Check current versions
pnpm list @digistratum/layout @digistratum/ds-core

# Update to latest
pnpm update @digistratum/layout @digistratum/ds-core
```

## Debugging Tips

### Enable verbose logging

```bash
# tools/cmd/create-app.sh
DEBUG=1 ./tools/cmd/create-app.sh -n my-app

# tools/cmd/update-from-template.sh (run from app directory)
cd ~/repos/digistratum/my-app
./tools/cmd/update-from-template.sh --dry-run

# Backend
LOG_LEVEL=debug go run cmd/api/main.go

# Frontend
DEBUG=* pnpm dev
```

### Check template version

```bash
# Your app's version
cat .template-version

# Latest template version
curl -s https://raw.githubusercontent.com/DigiStratum/ds-app-template/main/.template-version
```

### Validate configuration

```bash
# Check TypeScript config
npx tsc --noEmit

# Check Go module
go mod verify

# Validate CDK
cd infra && npx cdk synth
```

## Getting Help

1. Check this troubleshooting guide
2. Search existing GitHub issues in ds-app-template
3. Review recent changes in CHANGELOG.md
4. Create new issue with:
   - Template version (`.template-version`)
   - Error message
   - Steps to reproduce
   - Relevant logs

---

## Related Documentation

- [Deployment Guide](deployment.md) — CI/CD and environment setup
- [Getting Started](../getting-started.md) — Initial setup
