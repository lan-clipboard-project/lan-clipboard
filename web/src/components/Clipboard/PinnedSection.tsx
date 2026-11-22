/**
 * PinnedSection — displays pinned clips in a collapsible section at the top.
 */

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { ClipRow } from '../../hooks/useClipExits';
import ClipCard from './ClipCard';
import ClipPresence from './ClipPresence';

interface PinnedSectionProps {
  rows: ClipRow[];
  onPreviewImage: (url: string) => void;
  onRequestDelete: (id: string) => void;
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export default function PinnedSection({ rows, onPreviewImage, onRequestDelete, onShowToast }: PinnedSectionProps) {
  const { t } = useTranslation();
  const [collapsed, setCollapsed] = useState(false);

  const exiting = rows.every((row) => row.exiting);

  return (
    <ClipPresence exiting={exiting}>
      <div className="pb-6">
        {/* Header */}
        <button
          onClick={() => setCollapsed(!collapsed)}
          className="flex items-center gap-2 mb-3 text-sm font-semibold text-amber-600 dark:text-amber-400 hover:text-amber-700 dark:hover:text-amber-300"
        >
          <svg
            className={`w-4 h-4 transition-transform ${collapsed ? '' : 'rotate-90'}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
          </svg>
          <span>{t('date.pinned')} ({rows.length})</span>
        </button>

        {/* Pinned clips list */}
        {!collapsed && (
          <div className="flow-root">
            <div className="-mb-3">
              {rows.map(({ clip, exiting: clipExiting }) => (
                <ClipPresence key={clip.id} exiting={clipExiting}>
                  <div className="pb-3">
                    <ClipCard clip={clip} onPreviewImage={onPreviewImage} onRequestDelete={onRequestDelete}
                      onShowToast={onShowToast} />
                  </div>
                </ClipPresence>
              ))}
            </div>
          </div>
        )}
      </div>
    </ClipPresence>
  );
}
