# FR-THEME: Theming

> Light/dark theme support with user preference persistence.
> Theme is applied via CSS variables for instant switching without page reload.

---

## Requirements

### FR-THEME-001: Light and dark theme options

The application supports both light and dark visual themes.

**Acceptance Criteria:**
1. Light theme: white/light backgrounds, dark text
2. Dark theme: dark backgrounds, light text
3. Both themes meet WCAG 2.1 AA contrast requirements (4.5:1 for text)
4. Theme toggle accessible from user menu or settings
5. Theme changes apply immediately without page reload
6. All UI components render correctly in both themes (no hardcoded colors)

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `frontend/src/hooks/useTheme.test.tsx` |
| E2E test | `frontend/e2e/theme.spec.ts:toggle theme` |
| E2E test | `frontend/e2e/accessibility.spec.ts:contrast ratios` |
| Visual test | Chromatic snapshots in both themes |

**Evidence:**
- CI test results
- Lighthouse accessibility score ≥ 90 in both themes

---

### FR-THEME-002: Theme preference stored in user session

User's theme preference persists across sessions and devices.

**Acceptance Criteria:**
1. Theme preference stored in `ds_prefs` cookie (client-side)
2. Preference survives browser close/reopen
3. New session inherits last-used theme preference
4. Default theme follows system preference (`prefers-color-scheme`)
5. Explicit user choice overrides system preference
6. Theme preference syncs across DS apps (shared cookie domain)

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `frontend/src/hooks/useTheme.test.tsx:persists preference` |
| E2E test | `frontend/e2e/theme.spec.ts:preference persistence` |

**Evidence:**
- CI test results
- Manual: set dark mode → close browser → reopen → dark mode restored

---

### FR-THEME-003: Theme applied via CSS variables

Theming is implemented using CSS custom properties for maintainability and performance.

**Acceptance Criteria:**
1. Theme colors defined as CSS variables in `:root` and `[data-theme="dark"]`
2. Components use CSS variables, not hardcoded color values
3. Theme switch updates `data-theme` attribute on `<html>` element
4. No flash of incorrect theme on page load (critical CSS inlined)
5. Design tokens documented in `@digistratum/design-tokens`
6. Variables follow naming convention: `--ds-{category}-{name}`

**Verification:**
| Method | Location |
|--------|----------|
| Manual review | `frontend/src/styles/globals.css` |
| Unit test | `frontend/src/hooks/useTheme.test.tsx:applies data-theme` |
| Build check | Lint rule for hardcoded colors |

**Evidence:**
- CI lint results (no hardcoded colors)
- CSS inspection shows CSS variable usage

---

## Implementation

### CSS Variables

```css
/* frontend/src/styles/globals.css */
:root {
  /* Light theme (default) */
  --ds-background: #ffffff;
  --ds-foreground: #1a1a1a;
  --ds-muted: #f4f4f5;
  --ds-muted-foreground: #71717a;
  --ds-primary: #0066cc;
  --ds-primary-foreground: #ffffff;
  --ds-border: #e4e4e7;
  --ds-input: #e4e4e7;
  --ds-ring: #0066cc;
}

[data-theme="dark"] {
  --ds-background: #18181b;
  --ds-foreground: #fafafa;
  --ds-muted: #27272a;
  --ds-muted-foreground: #a1a1aa;
  --ds-primary: #3b82f6;
  --ds-primary-foreground: #ffffff;
  --ds-border: #27272a;
  --ds-input: #27272a;
  --ds-ring: #3b82f6;
}

body {
  background-color: var(--ds-background);
  color: var(--ds-foreground);
}
```

### Theme Hook

```tsx
// frontend/src/hooks/useTheme.tsx
type Theme = 'light' | 'dark' | 'system';

export function useTheme() {
  const [theme, setThemeState] = useState<Theme>(() => {
    // Read from cookie or default to system
    const stored = getCookie('ds_prefs_theme');
    return (stored as Theme) || 'system';
  });
  
  const resolvedTheme = useMemo(() => {
    if (theme === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches 
        ? 'dark' 
        : 'light';
    }
    return theme;
  }, [theme]);
  
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', resolvedTheme);
  }, [resolvedTheme]);
  
  const setTheme = (newTheme: Theme) => {
    setCookie('ds_prefs_theme', newTheme, { 
      domain: '.digistratum.com',
      expires: 365 
    });
    setThemeState(newTheme);
  };
  
  return { theme, resolvedTheme, setTheme };
}
```

### Prevent Flash of Wrong Theme

```html
<!-- In index.html <head> -->
<script>
  (function() {
    const stored = document.cookie.match(/ds_prefs_theme=([^;]+)/)?.[1];
    let theme = stored || 'system';
    if (theme === 'system') {
      theme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    document.documentElement.setAttribute('data-theme', theme);
  })();
</script>
```

### Theme Toggle Component

```tsx
// frontend/src/components/ThemeToggle.tsx
export function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" aria-label="Toggle theme">
          <SunIcon className="h-5 w-5 dark:hidden" />
          <MoonIcon className="h-5 w-5 hidden dark:block" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem onClick={() => setTheme('light')}>
          Light
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme('dark')}>
          Dark
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setTheme('system')}>
          System
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
```

---

## Design Tokens

Theme colors are defined in `@digistratum/design-tokens`:

| Token | Light | Dark | Usage |
|-------|-------|------|-------|
| `--ds-background` | #ffffff | #18181b | Page background |
| `--ds-foreground` | #1a1a1a | #fafafa | Primary text |
| `--ds-muted` | #f4f4f5 | #27272a | Secondary backgrounds |
| `--ds-primary` | #0066cc | #3b82f6 | Interactive elements |
| `--ds-destructive` | #dc2626 | #ef4444 | Error states |

---

## Traceability

| Requirement | Implementation | Test | Status |
|-------------|----------------|------|--------|
| FR-THEME-001 | `globals.css`, `useTheme.tsx` | `useTheme.test.tsx`, `theme.spec.ts` | ⚠️ |
| FR-THEME-002 | `useTheme.tsx`, cookie persistence | `useTheme.test.tsx`, `theme.spec.ts` | ⚠️ |
| FR-THEME-003 | CSS variables in `globals.css` | CSS lint, manual review | ⚠️ |

---

*Last updated: 2026-03-23*
