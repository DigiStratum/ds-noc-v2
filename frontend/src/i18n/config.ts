import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

// Minimal i18n configuration for DSAppShell
// Add HttpBackend and LanguageDetector if you need full i18n support
i18n
  .use(initReactI18next)
  .init({
    fallbackLng: 'en',
    supportedLngs: ['en'],
    
    interpolation: {
      escapeValue: false,
    },
    
    // No backend - DSAppShell uses inline fallbacks
    resources: {
      en: {
        translation: {}
      }
    },
  });

export default i18n;
