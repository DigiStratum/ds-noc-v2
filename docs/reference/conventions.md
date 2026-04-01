# Code Conventions

**This file is template-maintained.** Do not edit directly — changes will be overwritten by template updates.

## Overview

Consistent conventions across all DS apps enable agent autonomy and reduce cognitive load when switching between projects.

## Go (Backend)

### Package Structure
- `internal/` — Non-exported packages (app-specific logic)
- `pkg/` — Exported packages (reusable across apps)
- One handler per file in `internal/handlers/`

### Naming
- Handlers: `handle<Resource><Action>.go` (e.g., `handleUserGet.go`)
- Services: `<resource>.go` in `internal/services/`
- Storage: Interface in `internal/storage/`, implementations in subdirectories

### Patterns
- Use `HALResource` for all API responses (see [API Standards](api-standards.md))
- Never manually edit `main.go` for endpoints — use `go run ./tools/cmd/add-endpoint`
- Structured logging with request context

### Error Handling
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to fetch user %s: %w", userID, err)
}
```

### Testing
- Table-driven tests
- Test files co-located with source (`foo_test.go`)
- Use `testify/assert` for assertions

## TypeScript (Frontend)

### Component Structure
- Functional components with hooks (no class components)
- Co-locate tests with components (`Component.test.tsx`)
- App-specific code in `src/app/`
- Shared components in `src/components/`

### Naming
- Components: PascalCase (`UserProfile.tsx`)
- Hooks: `use` prefix (`useAuth.ts`)
- Utils: camelCase (`formatDate.ts`)

### Patterns
- Use `@digistratum/layout` for shell/chrome
- Props interfaces defined above component
- Destructure props in function signature

```tsx
interface UserCardProps {
  user: User;
  onEdit?: () => void;
}

export function UserCard({ user, onEdit }: UserCardProps) {
  // ...
}
```

### State Management
- React hooks for local state
- Context for app-wide state (auth, theme)
- No external state libraries unless justified

### Testing
- Vitest + React Testing Library
- Test behavior, not implementation
- Mock API calls, not components

## Infrastructure (CDK)

### Stack Structure
- One stack per app (`app-stack.ts`)
- Shared constructs in `constructs/`
- Environment via CDK context, not hardcoded

### Naming
- Stacks: `<AppName>Stack`
- Constructs: `<AppName><Resource>` (e.g., `MyAppApi`)

### Secrets
- Referenced by ARN, never embedded
- Use AWS Secrets Manager for all secrets
- No secrets in CDK context or environment variables

### Patterns
```typescript
// Good: Reference by ARN
const secret = Secret.fromSecretNameV2(this, 'DbSecret', 'myapp/db');

// Bad: Hardcoded value
const dbPassword = 'hunter2';
```

## Commit Messages

Format: `type(scope): description`

Types:
- `feat` — New feature
- `fix` — Bug fix
- `docs` — Documentation
- `refactor` — Code change (no behavior change)
- `test` — Test additions/changes
- `chore` — Build, tooling, deps

Examples:
```
feat(api): add user profile endpoint
fix(auth): handle expired session tokens
docs(readme): update deployment steps
```

## Related Documentation

- [API Standards](api-standards.md) — HAL/HATEOAS patterns
- [Testing Guide](testing.md) — Test requirements
- [Tech Stack](tech-stack.md) — Technologies used
