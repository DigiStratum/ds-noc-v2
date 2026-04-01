# NFR-A11Y: Accessibility Requirements

> Accessibility standards for DigiStratum applications.
> Target: WCAG 2.1 AA compliance. No user left behind.

---

## Audit-Ready Summary

### NFR-A11Y-001: WCAG 2.1 AA Compliance

**Acceptance Criteria:**
1. axe-core accessibility scan returns 0 violations (wcag2a, wcag2aa tags)
2. All perceivable criteria met (text alternatives, captions, contrast)
3. All operable criteria met (keyboard nav, focus visible, no traps)
4. All understandable criteria met (labels, error identification)
5. All robust criteria met (valid ARIA usage)

**Verification:**
| Method | Location |
|--------|----------|
| E2E test | `frontend/e2e/accessibility.spec.ts` (axe-core) |
| CI gate | `.github/workflows/ci.yml` (Playwright a11y tests) |
| Manual audit | Quarterly WCAG checklist review |

**Evidence:** axe-core scan results with 0 violations

---

### NFR-A11Y-002: Semantic HTML Structure

**Acceptance Criteria:**
1. Every page has exactly one `<h1>` element
2. Heading levels never skip (h1 → h2 → h3, not h1 → h3)
3. Navigation uses `<nav>` with `aria-label`
4. Main content in `<main>` element
5. Forms have associated `<label>` elements
6. Tables have `<caption>` and proper `scope` attributes

**Verification:**
| Method | Location |
|--------|----------|
| E2E test | `frontend/e2e/accessibility.spec.ts:heading hierarchy` |
| Manual review | Code review checklist |

**Evidence:** E2E test results, HTML validation

---

### NFR-A11Y-003: Keyboard Navigation Support

**Acceptance Criteria:**
1. All interactive elements reachable via Tab key
2. Focus order matches visual order (no jumps)
3. Focus indicator visible on all focusable elements (2px+ outline)
4. Escape key closes modals/dropdowns
5. No keyboard traps (focus can always escape)
6. Skip link available: "Skip to main content"

**Verification:**
| Method | Location |
|--------|----------|
| E2E test | `frontend/e2e/accessibility.spec.ts:keyboard navigation` |
| Manual test | Tab through entire app |

**Evidence:** E2E test results, manual keyboard testing video/checklist

---

### NFR-A11Y-004: Screen Reader Compatibility

**Acceptance Criteria:**
1. All images have descriptive `alt` text (or `alt=""` for decorative)
2. Icon-only buttons have `aria-label`
3. Dynamic content updates use `aria-live` regions
4. Modal dialogs have `role="dialog"` and `aria-modal="true"`
5. Error messages announced via `role="alert"`
6. Form inputs have accessible names via `label` or `aria-label`

**Verification:**
| Method | Location |
|--------|----------|
| E2E test | `frontend/e2e/accessibility.spec.ts` |
| Manual test | VoiceOver (macOS) or NVDA (Windows) |

**Evidence:** Screen reader testing checklist completed

---

### NFR-A11Y-005: Color Contrast Compliance

**Acceptance Criteria:**
1. Normal text: contrast ratio ≥ 4.5:1
2. Large text (≥18px or ≥14px bold): contrast ratio ≥ 3:1
3. UI components and graphics: contrast ratio ≥ 3:1
4. Information not conveyed by color alone
5. Focus indicators meet contrast requirements

**Verification:**
| Method | Location |
|--------|----------|
| axe-core | `frontend/e2e/accessibility.spec.ts` |
| Manual check | WebAIM Contrast Checker |

**Evidence:** axe-core color contrast checks pass

---

## Quick Reference

| Requirement | Standard | Description |
|-------------|----------|-------------|
| NFR-A11Y-001 | WCAG 2.1 AA | Full WCAG 2.1 Level AA compliance |
| NFR-A11Y-002 | Semantic HTML | Structure conveyed through markup |
| NFR-A11Y-003 | Keyboard Navigation | All functionality via keyboard |
| NFR-A11Y-004 | Screen Reader | Compatible with assistive tech |
| NFR-A11Y-005 | Color Contrast | WCAG contrast ratios met |

---

## WCAG 2.1 AA Compliance (NFR-A11Y-001)

WCAG 2.1 AA is our target conformance level, organized around four principles (POUR).

### 1. Perceivable

Users must be able to perceive information presented.

| Criterion | ID | Requirement | Target |
|-----------|-----|-------------|--------|
| Text alternatives | 1.1.1 | All non-text content has text alternatives | Required |
| Captions | 1.2.2 | Pre-recorded audio has captions | Required |
| Info and relationships | 1.3.1 | Structure conveyed through markup | Required |
| Meaningful sequence | 1.3.2 | Reading order matches visual order | Required |
| Sensory characteristics | 1.3.3 | Instructions don't rely solely on shape/location | Required |
| Orientation | 1.3.4 | Content not restricted to single orientation | Required |
| Input purpose | 1.3.5 | Input field purpose programmatically determined | Required |
| **Color contrast** | 1.4.3 | **4.5:1 ratio for text, 3:1 for large text** | Required |
| Resize text | 1.4.4 | Text resizable to 200% without loss | Required |
| Reflow | 1.4.10 | Content reflows at 320px without horizontal scroll | Required |
| Non-text contrast | 1.4.11 | UI components have 3:1 contrast ratio | Required |
| Text spacing | 1.4.12 | No loss of content when spacing adjusted | Required |
| Content on hover/focus | 1.4.13 | Additional content dismissible, hoverable | Required |

### 2. Operable

Users must be able to operate interface components.

| Criterion | ID | Requirement | Target |
|-----------|-----|-------------|--------|
| **Keyboard accessible** | 2.1.1 | **All functionality available via keyboard** | Required |
| No keyboard trap | 2.1.2 | Keyboard focus can always be moved away | Required |
| Timing adjustable | 2.2.1 | Time limits can be adjusted | Required |
| Pause, stop, hide | 2.2.2 | Moving content can be paused | Required |
| Skip links | 2.4.1 | Skip navigation mechanism provided | Required |
| Page titled | 2.4.2 | Pages have descriptive titles | Required |
| **Focus order** | 2.4.3 | **Focus order preserves meaning** | Required |
| Link purpose | 2.4.4 | Link purpose clear from text or context | Required |
| Multiple ways | 2.4.5 | Multiple ways to locate pages | Required |
| Headings and labels | 2.4.6 | Headings and labels describe purpose | Required |
| **Focus visible** | 2.4.7 | **Keyboard focus indicator visible** | Required |
| Pointer cancellation | 2.5.2 | Down-event doesn't trigger action | Required |
| Label in name | 2.5.3 | Accessible name includes visible label | Required |

### 3. Understandable

Users must be able to understand information and operation.

| Criterion | ID | Requirement | Target |
|-----------|-----|-------------|--------|
| Language of page | 3.1.1 | Page language specified | Required |
| Language of parts | 3.1.2 | Language of passages specified | Required |
| On focus | 3.2.1 | Focus doesn't trigger context change | Required |
| On input | 3.2.2 | Input doesn't trigger unexpected change | Required |
| Consistent navigation | 3.2.3 | Navigation consistent across pages | Required |
| **Error identification** | 3.3.1 | **Errors identified and described** | Required |
| Labels or instructions | 3.3.2 | Input fields have labels | Required |
| Error suggestion | 3.3.3 | Error correction suggestions provided | Required |

### 4. Robust

Content must be robust enough for assistive technologies.

| Criterion | ID | Requirement | Target |
|-----------|-----|-------------|--------|
| Name, role, value | 4.1.2 | Custom components expose name, role, value | Required |
| **Status messages** | 4.1.3 | **Status messages announced without focus** | Required |

---

## Semantic HTML (NFR-A11Y-002)

### Document Structure

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Page Title - App Name</title>
</head>
<body>
  <a href="#main" class="skip-link">Skip to main content</a>
  
  <header role="banner">
    <nav aria-label="Main navigation">
      <!-- Primary navigation -->
    </nav>
  </header>
  
  <main id="main" tabindex="-1">
    <h1>Page Heading</h1>
    <!-- Page content -->
  </main>
  
  <aside aria-label="Related content">
    <!-- Sidebar content -->
  </aside>
  
  <footer role="contentinfo">
    <!-- Footer content -->
  </footer>
</body>
</html>
```

### Heading Hierarchy

Always maintain proper heading levels. Never skip levels for styling.

```html
<!-- ✅ Correct -->
<h1>Dashboard</h1>
  <h2>Recent Activity</h2>
    <h3>Today</h3>
    <h3>This Week</h3>
  <h2>Quick Actions</h2>

<!-- ❌ Wrong - skips h2 -->
<h1>Dashboard</h1>
  <h3>Recent Activity</h3>
```

### Interactive Elements

Use native elements whenever possible.

```tsx
// ✅ Use native button
<button type="button" onClick={handleClick}>
  Click me
</button>

// ❌ Avoid div buttons
<div onClick={handleClick} className="button">
  Click me
</div>

// ✅ Use native link for navigation
<a href="/dashboard">Go to Dashboard</a>

// ❌ Avoid onClick for navigation
<span onClick={() => navigate('/dashboard')}>
  Go to Dashboard
</span>
```

### Forms

```tsx
<form onSubmit={handleSubmit}>
  <div>
    <label htmlFor="email">Email address</label>
    <input 
      type="email" 
      id="email" 
      name="email"
      required
      aria-describedby="email-hint email-error"
    />
    <span id="email-hint" className="hint">
      We'll never share your email
    </span>
    <span id="email-error" className="error" role="alert">
      {errors.email}
    </span>
  </div>
  
  <fieldset>
    <legend>Notification preferences</legend>
    <label>
      <input type="checkbox" name="email_notify" />
      Email notifications
    </label>
  </fieldset>
  
  <button type="submit">Save preferences</button>
</form>
```

### Tables

```html
<table>
  <caption>Monthly sales by region</caption>
  <thead>
    <tr>
      <th scope="col">Region</th>
      <th scope="col">Q1</th>
      <th scope="col">Q2</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <th scope="row">North</th>
      <td>$10,000</td>
      <td>$12,000</td>
    </tr>
  </tbody>
</table>
```

---

## Keyboard Navigation (NFR-A11Y-003)

All interactive elements must be keyboard accessible.

### Required Keyboard Support

| Key | Action |
|-----|--------|
| `Tab` | Move focus to next focusable element |
| `Shift + Tab` | Move focus to previous focusable element |
| `Enter` / `Space` | Activate button or link |
| `Escape` | Close modal, dropdown, or cancel action |
| `Arrow keys` | Navigate within components (menus, tabs, lists) |
| `Home` / `End` | Jump to first/last item in list |

### Focus Management

```css
/* Visible focus indicator - NEVER remove outline without replacement */
:focus-visible {
  outline: 2px solid var(--ds-focus-ring);
  outline-offset: 2px;
}

/* Skip link */
.skip-link {
  position: absolute;
  top: -40px;
  left: 0;
  padding: 8px;
  background: var(--ds-primary);
  color: white;
  z-index: 100;
}

.skip-link:focus {
  top: 0;
}
```

### Focus Trap for Modals

```tsx
function Modal({ isOpen, onClose, children }) {
  const modalRef = useRef<HTMLDivElement>(null);
  
  useEffect(() => {
    if (!isOpen) return;
    
    const modal = modalRef.current;
    const focusableElements = modal?.querySelectorAll(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    
    const firstElement = focusableElements?.[0] as HTMLElement;
    const lastElement = focusableElements?.[focusableElements.length - 1] as HTMLElement;
    
    // Focus first element on open
    firstElement?.focus();
    
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      
      if (e.key === 'Tab') {
        if (e.shiftKey && document.activeElement === firstElement) {
          e.preventDefault();
          lastElement?.focus();
        } else if (!e.shiftKey && document.activeElement === lastElement) {
          e.preventDefault();
          firstElement?.focus();
        }
      }
    };
    
    modal?.addEventListener('keydown', handleKeyDown);
    return () => modal?.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);
  
  if (!isOpen) return null;
  
  return (
    <div 
      ref={modalRef}
      role="dialog" 
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      {children}
    </div>
  );
}
```

### Custom Component Keyboard Patterns

**Dropdown Menu:**
```tsx
function DropdownMenu({ items }) {
  const [activeIndex, setActiveIndex] = useState(0);
  
  const handleKeyDown = (e: KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setActiveIndex(i => Math.min(i + 1, items.length - 1));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setActiveIndex(i => Math.max(i - 1, 0));
        break;
      case 'Home':
        e.preventDefault();
        setActiveIndex(0);
        break;
      case 'End':
        e.preventDefault();
        setActiveIndex(items.length - 1);
        break;
    }
  };
  
  return (
    <ul role="menu" onKeyDown={handleKeyDown}>
      {items.map((item, index) => (
        <li 
          key={item.id}
          role="menuitem"
          tabIndex={index === activeIndex ? 0 : -1}
        >
          {item.label}
        </li>
      ))}
    </ul>
  );
}
```

---

## Screen Reader Support (NFR-A11Y-004)

### ARIA Attributes

| Attribute | Purpose | Example |
|-----------|---------|---------|
| `aria-label` | Provides accessible name | `<button aria-label="Close">×</button>` |
| `aria-labelledby` | References visible label | `<div aria-labelledby="section-title">` |
| `aria-describedby` | References description | `<input aria-describedby="hint error">` |
| `aria-hidden` | Hides from assistive tech | `<span aria-hidden="true">★</span>` |
| `aria-expanded` | Expandable state | `<button aria-expanded="false">Menu</button>` |
| `aria-pressed` | Toggle button state | `<button aria-pressed="true">Bold</button>` |
| `aria-selected` | Selection state | `<li role="option" aria-selected="true">` |
| `aria-current` | Current item in set | `<a aria-current="page">Dashboard</a>` |
| `aria-live` | Announces dynamic content | `<div aria-live="polite">` |
| `aria-busy` | Loading state | `<div aria-busy="true">Loading...</div>` |
| `aria-invalid` | Invalid input state | `<input aria-invalid="true">` |

### Live Regions

```tsx
// Status updates (polite)
<div role="status" aria-live="polite">
  3 items selected
</div>

// Error alerts (assertive)
<div role="alert">
  Form submission failed. Please try again.
</div>

// Loading states
<div aria-busy="true" aria-live="polite">
  Loading...
</div>
```

### Icon Buttons

```tsx
// Icon-only button - MUST have aria-label
<button aria-label="Close dialog" onClick={onClose}>
  <XIcon aria-hidden="true" />
</button>

// Toggle button with state
<button 
  aria-pressed={isBold}
  aria-label="Bold"
  onClick={() => setIsBold(!isBold)}
>
  <BoldIcon aria-hidden="true" />
</button>
```

### Modal Dialog

```tsx
function Dialog({ isOpen, onClose, title, children }) {
  const titleId = useId();
  
  return (
    <div 
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
    >
      <h2 id={titleId}>{title}</h2>
      {children}
      <button onClick={onClose}>Close</button>
    </div>
  );
}
```

### Tabs

```tsx
function Tabs({ tabs, activeIndex, onChange }) {
  return (
    <div>
      <div role="tablist" aria-label="Content tabs">
        {tabs.map((tab, index) => (
          <button
            key={tab.id}
            role="tab"
            id={`tab-${tab.id}`}
            aria-selected={index === activeIndex}
            aria-controls={`panel-${tab.id}`}
            tabIndex={index === activeIndex ? 0 : -1}
          >
            {tab.label}
          </button>
        ))}
      </div>
      
      {tabs.map((tab, index) => (
        <div
          key={tab.id}
          role="tabpanel"
          id={`panel-${tab.id}`}
          aria-labelledby={`tab-${tab.id}`}
          hidden={index !== activeIndex}
          tabIndex={0}
        >
          {tab.content}
        </div>
      ))}
    </div>
  );
}
```

---

## Color and Contrast

### Minimum Contrast Ratios

| Element Type | Minimum Ratio | Tool |
|--------------|---------------|------|
| Normal text (< 18px) | 4.5:1 | WebAIM Contrast Checker |
| Large text (≥ 18px or ≥ 14px bold) | 3:1 | WebAIM Contrast Checker |
| UI components | 3:1 | WebAIM Contrast Checker |
| Graphics | 3:1 | WebAIM Contrast Checker |

### Color-Independent Information

```tsx
// ❌ Wrong - relies on color alone
<span style={{ color: 'red' }}>Error</span>

// ✅ Correct - includes icon/text
<span role="alert" style={{ color: 'red' }}>
  <ErrorIcon aria-hidden="true" /> Error: Invalid email
</span>
```

### CSS Custom Properties

```css
:root {
  /* Ensure all color pairs meet contrast requirements */
  --ds-text-primary: #1a1a1a;      /* On white: 16:1 */
  --ds-text-secondary: #555555;     /* On white: 7:1 */
  --ds-text-on-primary: #ffffff;    /* On primary: 4.5:1+ */
  --ds-error: #c41e3a;             /* On white: 5.2:1 */
  --ds-focus-ring: #0066cc;        /* Visible on all backgrounds */
}

/* Dark mode equivalents */
[data-theme="dark"] {
  --ds-text-primary: #f0f0f0;
  --ds-text-secondary: #b0b0b0;
  --ds-background: #1a1a1a;
}
```

---

## Testing Approach

### Automated Testing

```typescript
// E2E accessibility testing with axe-core
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.describe('Accessibility', () => {
  test('homepage has no violations', async ({ page }) => {
    await page.goto('/');
    
    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();
    
    expect(results.violations).toEqual([]);
  });

  test('all pages have proper heading hierarchy', async ({ page }) => {
    await page.goto('/');
    
    const headings = await page.$$eval('h1, h2, h3, h4, h5, h6', (elements) =>
      elements.map(el => ({ tag: el.tagName, text: el.textContent }))
    );
    
    // Verify h1 exists and is first
    expect(headings[0]?.tag).toBe('H1');
  });

  test('focus is visible on interactive elements', async ({ page }) => {
    await page.goto('/');
    
    // Tab to first interactive element
    await page.keyboard.press('Tab');
    
    const focusedElement = await page.evaluate(() => {
      const el = document.activeElement;
      const styles = window.getComputedStyle(el as Element);
      return {
        outline: styles.outline,
        boxShadow: styles.boxShadow,
      };
    });
    
    // Verify focus indicator exists
    expect(
      focusedElement.outline !== 'none' || 
      focusedElement.boxShadow !== 'none'
    ).toBe(true);
  });
});
```

### Manual Testing Checklist

**Keyboard Navigation:**
- [ ] Can navigate all interactive elements with Tab
- [ ] Can activate buttons with Enter and Space
- [ ] Can close modals with Escape
- [ ] Focus order matches visual order
- [ ] Focus indicator is always visible

**Screen Reader:**
- [ ] All images have alt text
- [ ] Form inputs have associated labels
- [ ] Buttons have accessible names
- [ ] Live regions announce updates
- [ ] Modals are announced as dialogs

**Visual:**
- [ ] Text meets contrast requirements
- [ ] Information not conveyed by color alone
- [ ] Content readable at 200% zoom
- [ ] No horizontal scroll at 320px width

### Tools

| Tool | Purpose |
|------|---------|
| axe DevTools | Browser extension for a11y testing |
| WAVE | Web accessibility evaluation |
| Lighthouse | Automated accessibility audit |
| VoiceOver (macOS) | Screen reader testing |
| NVDA (Windows) | Screen reader testing |
| Color Contrast Checker | Verify color ratios |

---

## Implementation Checklist

### Component Level

- [ ] Use semantic HTML elements
- [ ] Provide accessible names for interactive elements
- [ ] Implement keyboard support for custom widgets
- [ ] Use ARIA attributes appropriately
- [ ] Test with screen reader

### Page Level

- [ ] Include skip link
- [ ] Set page language
- [ ] Use proper heading hierarchy
- [ ] Provide page title
- [ ] Ensure focus management on navigation

### Application Level

- [ ] Document accessibility patterns
- [ ] Include a11y tests in CI
- [ ] Train team on accessibility
- [ ] Conduct regular audits

---

## Traceability

| Requirement | Implementation | Test |
|-------------|----------------|------|
| NFR-A11Y-001 | WCAG compliance | axe-core E2E tests |
| NFR-A11Y-002 | Semantic HTML | Heading hierarchy tests |
| NFR-A11Y-003 | Keyboard navigation | Tab order E2E tests |
| NFR-A11Y-004 | Screen reader | ARIA attribute tests |

---

*Last updated: 2026-03-22*
