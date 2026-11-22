/**
 * SearchBar — expandable search input that slides down from header.
 * Features debounced search and type filter tabs.
 */

import { useEffect, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useClipStore } from '../../store/clipStore';
import { SEARCH_DEBOUNCE_MS } from '../../utils/constants';
import type { ClipType } from '../../types';

interface SearchBarProps {
  onClose: () => void;
}

export default function SearchBar({ onClose }: SearchBarProps) {
  const { t } = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>();

  const searchQuery = useClipStore((s) => s.searchQuery);
  const filterType = useClipStore((s) => s.filterType);
  const setSearch = useClipStore((s) => s.setSearch);
  const performSearch = useClipStore((s) => s.performSearch);
  const setFilter = useClipStore((s) => s.setFilter);
  const fetchClips = useClipStore((s) => s.fetchClips);

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  // Debounced search
  const handleInputChange = useCallback(
    (value: string) => {
      setSearch(value);
      if (debounceRef.current) clearTimeout(debounceRef.current);
      debounceRef.current = setTimeout(() => {
        if (value.trim()) {
          performSearch();
        }
      }, SEARCH_DEBOUNCE_MS);
    },
    [setSearch, performSearch],
  );

  // Escape to close
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Escape') {
        setSearch('');
        onClose();
      }
    },
    [onClose, setSearch],
  );

  // Type filter handler
  const handleFilterChange = useCallback(
    (type: ClipType | null) => {
      setFilter(type);
      // Keep the timeline in sync with the filter when returning from search.
      fetchClips();
      if (searchQuery.trim()) {
        setTimeout(() => performSearch(), 0);
      }
    },
    [setFilter, searchQuery, performSearch, fetchClips],
  );

  const filterOptions: { key: ClipType | null; label: string }[] = [
    { key: null, label: t('search.all') },
    { key: 'text', label: t('search.text') },
    { key: 'image', label: t('search.image') },
    { key: 'file', label: t('search.file') },
  ];

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 animate-slide-down">
      <div className="max-w-3xl mx-auto px-4 py-3 space-y-2">
        {/* Search input */}
        <div className="relative">
          <svg
            className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            ref={inputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => handleInputChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t('header.search')}
            className="w-full pl-10 pr-8 py-2 text-sm bg-gray-50 dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent placeholder-gray-400 dark:placeholder-gray-500"
          />
          {searchQuery && (
            <button
              onClick={() => {
                setSearch('');
                inputRef.current?.focus();
              }}
              className="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded-md hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-400"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>

        {/* Type filter tabs */}
        <div className="flex gap-1">
          {filterOptions.map((opt) => (
            <button
              key={opt.key ?? 'all'}
              onClick={() => handleFilterChange(opt.key)}
              className={`px-3 py-1 text-xs rounded-full ${
                filterType === opt.key
                  ? 'bg-blue-500 text-white'
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-600'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
