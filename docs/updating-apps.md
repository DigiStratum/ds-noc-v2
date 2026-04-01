# Updating Apps from Template

Guide to receiving template updates in your derived app without losing business logic.

## Template Versioning

The template uses **semantic versioning** (MAJOR.MINOR.PATCH):

| Change Type | Version Bump | Example |
|-------------|--------------|---------|
| Breaking changes (API, structure) | MAJOR | 0.x → 1.0 |
| New features, middleware, tools | MINOR | 0.7 → 0.8 |
| Bug fixes, docs, small tweaks | PATCH | 0.7.0 → 0.7.1 |

**Every template change must bump the version.** The version lives in `.template-version`.

### For Template Maintainers

When committing changes to ds-app-template:

1. Determine version bump type based on change
2. Update `.template-version` before committing
3. Include version in commit message: `feat: add txnlog middleware (v0.8.0)`

## Overview

The `go run ./tools/cmd/update-from-template` script applies template improvements to your app while preserving your app-owned files.

**What gets updated** (defined in `.template-manifest`):
- Infrastructure files (backend middleware, frontend components, CDK)
- Configuration files (package.json, tsconfig, etc.)
- CI/CD workflows
- Documentation
- `.template-version` marker

**What is NEVER touched** (app-owned):
- `backend/internal/handlers/` — Your API handlers
- `frontend/src/app/` — Your pages, features, config
- `README.md` — Your app description
- `.git/` directory
- Environment files (`.env*`)
- Files listed in your app's `.template-overrides`

## Quick Update

Run from your derived app directory:

```bash
cd ~/repos/digistratum/my-app
go run ./tools/cmd/update-from-template
```

The script auto-detects the template repo location. If needed, specify it:

```bash
go run ./tools/cmd/update-from-template --template-path ~/repos/digistratum/ds-app-template
```

## Step-by-Step Process

### 1. Check Current Version

```bash
cat .template-version
```

### 2. Preview Changes (Dry Run)

Always preview before applying:

```bash
go run ./tools/cmd/update-from-template --dry-run
```

This shows:
- `[new]` — Files that will be added
- `[update]` — Files that will be modified  
- `[unchanged]` — Files already in sync
- `[override]` — Files skipped (listed in `.template-overrides`)

### 3. Apply Update

```bash
go run ./tools/cmd/update-from-template
```

### 4. Review Changes

```bash
git status
git diff
```

### 5. Test Locally

```bash
pnpm install
pnpm dev
pnpm test
cd backend && go test ./...
```

### 6. Commit

```bash
git add -A
git commit -m "chore: update from template v$(cat .template-version)"
```

## Command-Line Options

| Option | Description |
|--------|-------------|
| `--dry-run` | Preview changes without applying |
| `--template-path PATH` | Specify template repo location |
| `-h, --help` | Show help |

## How It Works

1. **Reads `.template-manifest`** from the template repo (authoritative file list)
2. **Reads `.template-overrides`** from your app repo (files to skip)
3. **Syncs each file individually**: manifest − overrides
4. **Updates `.template-version`** to match template

The manifest defines exactly which files are template-owned. See `.template-manifest` in the template repo for the full list.

## Customizing Template Files (`.template-overrides`)

Sometimes you need to customize a file that's normally template-owned. Instead of losing your changes on every update, add the file to `.template-overrides` in your app repo:

```bash
# Create or edit .template-overrides in your app root
cat >> .template-overrides << 'EOF'
# API layer - using custom client
frontend/src/api/client.ts
frontend/src/api/index.ts

# Custom auth flow
frontend/src/components/AuthShell.tsx
EOF
```

**Format:**
- One file path per line (exact paths, no globs)
- Comments start with `#`
- Paths are relative to repo root

**When to use:**
- You've customized a shared component for your app's needs
- You need app-specific API client configuration
- Template hooks/utilities don't fit your use case

**Dry run shows overrides:**
```
[INFO] App overrides: 3 files excluded
  [override] frontend/src/api/client.ts (skipped - app customized)
  [override] frontend/src/api/index.ts (skipped - app customized)
```

**Best practice:** Only override files you've actually modified. Check the template's version of the file occasionally to see if improvements should be merged manually.

## Multiple Apps

When maintaining multiple apps from the same template:

```bash
for app in ~/repos/digistratum/ds-*; do
  if [[ -f "$app/.template-version" && -f "$apgo run ./tools/cmd/update-from-template" ]]; then
    echo "Updating $app..."
    (cd "$app" && go run ./tools/cmd/update-from-template)
  fi
done
```

## Reverting Updates

If an update causes issues:

```bash
# Revert via git
git checkout HEAD~1 -- .

# Or selectively revert specific paths
git checkout HEAD~1 -- frontend/package.json
```

## Best Practices

### DO:
- Always run `--dry-run` first
- Test locally before committing
- Keep app-owned code in designated locations
- Update regularly to avoid large diffs

### DON'T:
- Modify template-owned files directly (changes will be overwritten)
- Skip testing after updates
- Ignore build/test failures

## Troubleshooting

### "Cannot find ds-app-template repo"

Specify the template location:
```bash
go run ./tools/cmd/update-from-template --template-path /path/to/ds-app-template
```

Or clone it to a standard location:
```bash
git clone https://github.com/DigiStratum/ds-app-template.git ~/repos/digistratum/ds-app-template
```

### Build fails after update

1. Clear dependencies: `rm -rf node_modules && pnpm install`
2. Clear Go cache: `go clean -modcache && go mod download`
3. Check for breaking changes in template CHANGELOG

### My changes to template files were lost

Template-owned files are replaced on update. You have two options:

**Option 1: Add to `.template-overrides`** (recommended for intentional customizations)
```bash
echo "frontend/src/api/client.ts" >> .template-overrides
```

**Option 2: Move to app-owned locations** (for new functionality)
- Custom components → `frontend/src/app/components/`
- Custom middleware → `backend/internal/handlers/` (or create new internal package)
- Custom config → Override in app-specific config files
