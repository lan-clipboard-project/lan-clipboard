/**
 * LanguageSwitch — toggles the UI language between Chinese and English.
 * Persists the choice to localStorage using the same key i18next detects on startup.
 */

import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { LS_LANG_KEY } from '../../utils/constants';

export default function LanguageSwitch() {
  const { t, i18n } = useTranslation();
  const nextLang = i18n.language.startsWith('zh') ? 'en' : 'zh';

  const handleToggle = useCallback(() => {
    void i18n.changeLanguage(nextLang);
    localStorage.setItem(LS_LANG_KEY, nextLang);
  }, [i18n, nextLang]);

  return (
    <button
      type="button"
      onClick={handleToggle}
      className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-600 dark:text-gray-400"
      title={t('language.switch')}
      aria-label={t('language.switch')}
    >
      <span className="block w-5 text-sm font-medium leading-5 text-center">
        {nextLang === 'zh' ? '中' : 'EN'}
      </span>
    </button>
  );
}
