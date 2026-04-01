# Template Overrides Audit

> Completed: 2026-03-27 (Issue #1809)

## Summary

The `.template-overrides` mechanism is only needed for files that:
1. **Exist in the template** (listed in `.template-manifest`)
2. **Have been customized** by the app beyond what token substitution handles

Files that are app-only (not in the template) don't need protection — the template won't overwrite them.

## Cleanup Completed

Removed unnecessary entries from all apps' `.template-overrides`:

### ds-kanban-v2
**Removed:** `frontend/src/api/client.ts`, `frontend/src/hooks/useProject.tsx`, `frontend/src/hooks/useWorkflow.ts`  
**Kept:** `frontend/src/api/index.ts`, `frontend/src/hooks/index.ts`

### dsaccount-v2
**Removed:** `frontend/src/api/auth.ts`, `frontend/src/api/auth.test.ts`, `frontend/src/components/AuthShell.tsx`, all `backend/internal/dynamo/*`  
**Kept:** `frontend/src/api/index.ts`, `frontend/src/components/index.ts`

### ds-app-workforce-v2
**Removed:** `frontend/src/api/client.ts`, all `backend/internal/*`, `frontend/src/vite-env.d.ts`, `go.work`  
**Kept:** `frontend/src/api/index.ts`

## The Index File Pattern

The remaining overrides are all `index.ts` files that need to export both template modules and app-specific modules. This is a structural issue, not a customization.

### Why Index Files Need Overrides

**Template `frontend/src/api/index.ts`:**
```typescript
export * from './hal';
```

**App adds its own modules:**
```typescript
export * from './hal';
export { api } from './client';  // app-specific
```

The app's version must include the template's exports plus app-specific exports.

### Future: Auto-merge Index Files

The `update-from-template` tool could be enhanced to:
1. Parse existing index.ts exports
2. Parse template's index.ts exports  
3. Merge (union of exports)
4. Write combined result

This would eliminate the need for index.ts overrides entirely.

### Current Workaround

Until auto-merge is implemented, apps must:
1. List their index.ts files in `.template-overrides`
2. Manually merge any new template exports when updating

## Guidelines

**Do NOT add to `.template-overrides`:**
- Files that don't exist in `.template-manifest`
- Files that only differ by token values (let token substitution handle it)
- Build outputs or generated files

**DO add to `.template-overrides`:**
- Template files that need structural customization
- Index files that export app-specific modules alongside template modules
