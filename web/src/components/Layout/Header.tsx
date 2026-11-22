/**
 * Header component — fixed at top.
 * Contains: app title, search toggle, theme toggle, language switch, clear-all button.
 */

import { forwardRef, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { ThemeMode } from '../../types';
import { useClipStore } from '../../store/clipStore';
import { deleteAllClips } from '../../api/clips';
import SearchBar from '../Search/SearchBar';
import ThemeToggle from '../Common/ThemeToggle';
import LanguageSwitch from '../Common/LanguageSwitch';
import ConfirmDialog from '../Common/ConfirmDialog';

interface HeaderProps {
  themeMode: ThemeMode;
  onThemeToggle: () => void;
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export default forwardRef<HTMLElement, HeaderProps>(function Header({ themeMode, onThemeToggle, onShowToast }, ref) {
  const { t } = useTranslation();
  const [showSearch, setShowSearch] = useState(false);
  const [showClearConfirm, setShowClearConfirm] = useState(false);

  const clearClips = useClipStore((s) => s.clearClips);
  const setSearch = useClipStore((s) => s.setSearch);

  const handleCloseSearch = useCallback(() => {
    setShowSearch(false);
    // Closing search must also leave search mode, otherwise the list keeps
    // showing stale search results with no visible search bar.
    setSearch('');
  }, [setSearch]);

  const handleClearAll = useCallback(async () => {
    try {
      await deleteAllClips();
      clearClips();
      onShowToast(t('dialog.cleared'), 'success');
    } catch {
      onShowToast(t('error.clearFailed'), 'error');
    }
  }, [clearClips, onShowToast, t]);

  return (
    <>
      <header ref={ref} className="relative z-30 shrink-0 border-b frosted-bar">
        <div className="max-w-3xl mx-auto px-4 py-3 flex items-center justify-between">
          {/* Left: Logo + Title */}
          <div className="flex items-center gap-2 min-w-0">
            <img src="/assets/lan-clipboard.svg" alt="" width={28} height={28} className="shrink-0" />
            <h1 className="text-lg font-bold truncate">{t('header.title')}</h1>
          </div>

          {/* Right: Actions */}
          <div className="flex items-center gap-2 flex-shrink-0">
            {/* Search toggle */}
            <button
              onClick={() => setShowSearch(!showSearch)}
              className={`p-2 rounded-lg ${
                showSearch
                  ? 'bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400'
                  : 'hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-600 dark:text-gray-400'
              }`}
              title={t('header.search')}
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </button>

            {/* Theme toggle */}
            <ThemeToggle mode={themeMode} onToggle={onThemeToggle} />

            {/* Language switch */}
            <LanguageSwitch />

            {/* Clear all */}
            <button
              onClick={() => setShowClearConfirm(true)}
              className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-600 dark:text-gray-400"
              title={t('header.clearAll')}
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>

        {/* Search bar (expandable) */}
        {showSearch && <SearchBar onClose={handleCloseSearch} />}
      </header>

      {/* Clear all confirmation dialog */}
      {showClearConfirm && (
        <ConfirmDialog
          message={t('dialog.confirmClearAll')}
          onConfirm={handleClearAll}
          onClose={() => setShowClearConfirm(false)}
        />
      )}
    </>
  );
});
