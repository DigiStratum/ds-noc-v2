# FR-I18N: Internationalization

> Multi-language support with user preference persistence.
> Currently supports English; framework ready for additional languages.

---

## Requirements

### FR-I18N-001: Static strings loaded from language packs

UI text is externalized to language files for translation support.

**Acceptance Criteria:**
1. All user-visible strings use translation keys, not hardcoded text
2. Language files stored in `frontend/src/i18n/locales/{lang}.json`
3. English (`en`) is the default and fallback language
4. Missing translation keys fall back to English gracefully
5. Translation keys follow namespace pattern: `{namespace}.{key}` (e.g., `nav.home`)
6. Pluralization supported for count-based strings
7. Variable interpolation supported: `"welcome": "Hello, {{name}}!"`

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `frontend/src/i18n/config.test.ts` |
| E2E test | `frontend/e2e/i18n.spec.ts:loads translations` |
| Build check | Lint rule for hardcoded UI strings |

**Evidence:**
- CI test results
- Translation key audit script output

---

### FR-I18N-002: Dynamic content translated and cached on-the-fly

User-generated and dynamic content can be translated via translation service.

**Acceptance Criteria:**
1. Translation API available for dynamic content
2. Translations cached to avoid repeated API calls
3. Cache TTL configurable (default: 24 hours)
4. Untranslated content shows original text (no blank/error)
5. Translation failures logged but don't break UI
6. Rate limiting on translation API to control costs

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `frontend/src/i18n/translator.test.ts` |
| Integration test | `backend/test/integration/translation_test.go` |

**Evidence:**
- CI test results
- CloudWatch metrics showing translation cache hit rate

**Status:** ❌ Not implemented (Phase 2)

---

### FR-I18N-003: Language preference stored in user session

User's language preference persists across sessions.

**Acceptance Criteria:**
1. Language preference stored in `ds_prefs` cookie
2. Preference survives browser close/reopen
3. New session inherits last-used language preference
4. Default language follows browser `Accept-Language` header
5. Explicit user choice overrides browser preference
6. Language preference syncs across DS apps (shared cookie domain)
7. Language selector available in user menu or settings

**Verification:**
| Method | Location |
|--------|----------|
| Unit test | `frontend/src/hooks/useLanguage.test.tsx` |
| E2E test | `frontend/e2e/i18n.spec.ts:preference persistence` |

**Evidence:**
- CI test results
- Manual: set German → close browser → reopen → German restored

---

## Implementation

### i18n Configuration

```typescript
// frontend/src/i18n/config.ts
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import en from './locales/en.json';
// import de from './locales/de.json';  // Future

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      // de: { translation: de },
    },
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false, // React already escapes
    },
    detection: {
      order: ['cookie', 'navigator'],
      caches: ['cookie'],
      cookieDomain: '.digistratum.com',
    },
  });

export default i18n;
```

### Language File Structure

```json
// frontend/src/i18n/locales/en.json
{
  "common": {
    "loading": "Loading...",
    "error": "An error occurred",
    "save": "Save",
    "cancel": "Cancel",
    "delete": "Delete",
    "confirm": "Confirm"
  },
  "nav": {
    "home": "Home",
    "dashboard": "Dashboard",
    "settings": "Settings",
    "logout": "Logout"
  },
  "auth": {
    "login": "Log in",
    "logout": "Log out",
    "welcome": "Welcome, {{name}}!"
  },
  "errors": {
    "notFound": "Page not found",
    "unauthorized": "Please log in to continue",
    "serverError": "Something went wrong. Please try again."
  }
}
```

### Using Translations

```tsx
// In components
import { useTranslation } from 'react-i18next';

function Header() {
  const { t } = useTranslation();
  
  return (
    <nav>
      <a href="/">{t('nav.home')}</a>
      <a href="/dashboard">{t('nav.dashboard')}</a>
    </nav>
  );
}

// With interpolation
function Welcome({ user }) {
  const { t } = useTranslation();
  return <h1>{t('auth.welcome', { name: user.name })}</h1>;
}

// With pluralization
// In JSON: "items": "{{count}} item", "items_plural": "{{count}} items"
function ItemCount({ count }) {
  const { t } = useTranslation();
  return <span>{t('items', { count })}</span>;
}
```

### Language Hook

```tsx
// frontend/src/hooks/useLanguage.tsx
export function useLanguage() {
  const { i18n } = useTranslation();
  
  const setLanguage = (lang: string) => {
    i18n.changeLanguage(lang);
    // Cookie is set automatically by LanguageDetector
  };
  
  return {
    language: i18n.language,
    setLanguage,
    availableLanguages: ['en'], // Expand as languages added
  };
}
```

### Language Selector Component

```tsx
// frontend/src/components/LanguageSelector.tsx
export function LanguageSelector() {
  const { language, setLanguage, availableLanguages } = useLanguage();
  
  const languageNames: Record<string, string> = {
    en: 'English',
    de: 'Deutsch',
    es: 'Español',
    fr: 'Français',
  };
  
  return (
    <select 
      value={language} 
      onChange={(e) => setLanguage(e.target.value)}
      aria-label="Select language"
    >
      {availableLanguages.map((lang) => (
        <option key={lang} value={lang}>
          {languageNames[lang] || lang}
        </option>
      ))}
    </select>
  );
}
```

---

## Supported Languages

| Language | Code | Status |
|----------|------|--------|
| English | `en` | ✅ Default |
| German | `de` | ❌ Planned |
| Spanish | `es` | ❌ Planned |
| French | `fr` | ❌ Planned |

---

## Translation Workflow

1. **Adding new strings:**
   - Add key to `en.json`
   - Use `t('namespace.key')` in component
   - Add translations to other language files (when available)

2. **Auditing for hardcoded strings:**
   ```bash
   # Find potential hardcoded strings
   grep -r ">[A-Z][a-z]" src/components/ --include="*.tsx" | grep -v "{t("
   ```

3. **Validating translation files:**
   - CI check compares all locale files for missing keys
   - Missing keys logged as warnings (not build failures)

---

## Traceability

| Requirement | Implementation | Test | Status |
|-------------|----------------|------|--------|
| FR-I18N-001 | `frontend/src/i18n/config.ts`, locale JSON files | `config.test.ts`, `i18n.spec.ts` | ⚠️ |
| FR-I18N-002 | Not implemented | - | ❌ |
| FR-I18N-003 | `useLanguage.tsx`, i18next LanguageDetector | `useLanguage.test.tsx`, `i18n.spec.ts` | ⚠️ |

---

*Last updated: 2026-03-23*
