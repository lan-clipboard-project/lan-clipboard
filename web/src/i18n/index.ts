import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import zh from './zh.json';
import en from './en.json';
import { LS_LANG_KEY } from '../utils/constants';

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      zh: { translation: zh },
      en: { translation: en },
    },
    fallbackLng: 'en',
    lng: localStorage.getItem(LS_LANG_KEY) || 'zh',
    interpolation: {
      escapeValue: false, // React already handles XSS
    },
    detection: {
      lookupLocalStorage: LS_LANG_KEY,
      caches: ['localStorage'],
    },
  });

// Keep <html lang> in sync with the active language (a11y + correct fonts).
i18n.on('languageChanged', (lng) => {
  document.documentElement.lang = lng;
});
document.documentElement.lang = i18n.language || 'zh';

export default i18n;
