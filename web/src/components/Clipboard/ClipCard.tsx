/**
 * ClipCard — renders a single clip entry.
 * Delegates content display to TextClip, ImageClip, or FileClip.
 * Provides pin/unpin and delete actions.
 */

import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { Clip } from '../../types';
import { updateClip } from '../../api/clips';
import { useClipStore } from '../../store/clipStore';
import { formatTime } from '../../utils/format';
import TextClip from './TextClip';
import ImageClip from './ImageClip';
import FileClip from './FileClip';

interface ClipCardProps {
  clip: Clip;
  onPreviewImage: (url: string) => void;
  onRequestDelete: (id: string) => void;
  onShowToast: (message: string, type: 'success' | 'error') => void;
  /** Search terms to highlight (only set while rendering search results). */
  highlightTerms?: string[];
}

export default function ClipCard({ clip, onPreviewImage, onRequestDelete, onShowToast, highlightTerms }: ClipCardProps) {
  const { t } = useTranslation();
  const updateClipInStore = useClipStore((s) => s.updateClip);

  const handleTogglePin = useCallback(async () => {
    try {
      const updated = await updateClip(clip.id, !clip.pinned);
      updateClipInStore(updated);
    } catch {
      onShowToast(t('error.updateFailed'), 'error');
    }
  }, [clip.id, clip.pinned, updateClipInStore, onShowToast, t]);

  return (
    <>
      <div className="clip-card group relative rounded-xl border animate-fade-in-up">
        {/* Content */}
        <div className="p-4">
          {clip.type === 'text' && (
            <TextClip clip={clip} onShowToast={onShowToast} highlightTerms={highlightTerms} />
          )}
          {clip.type === 'image' && (
            <ImageClip
              clip={clip}
              onPreviewImage={onPreviewImage}
              highlightTerms={highlightTerms}
            />
          )}
          {clip.type === 'file' && <FileClip clip={clip} highlightTerms={highlightTerms} />}
        </div>

        {/* Footer: timestamp, uploader and actions */}
        <div className="flex items-center justify-between px-4 pb-3 pt-0">
          <div className="flex flex-wrap items-center gap-x-2 text-xs text-gray-500 dark:text-gray-400">
            <time dateTime={clip.createdAt}>{formatTime(clip.createdAt)}</time>
            <span aria-hidden="true">·</span>
            <span>{clip.uploader || t('clip.unknownUploader')}</span>
          </div>

          <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 sm:opacity-100 transition-opacity">
            {/* Pin/Unpin */}
            <button
              onClick={handleTogglePin}
              className={`p-1.5 rounded-md text-xs ${
                clip.pinned
                  ? 'text-amber-500 hover:bg-amber-50 dark:hover:bg-amber-900/30'
                  : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 hover:text-amber-500'
              }`}
              title={clip.pinned ? t('clip.unpin') : t('clip.pin')}
            >
              <svg className="w-4 h-4" fill={clip.pinned ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
              </svg>
            </button>

            {/* Delete */}
            <button
              onClick={() => onRequestDelete(clip.id)}
              className="p-1.5 rounded-md text-gray-400 hover:bg-red-50 dark:hover:bg-red-900/30 hover:text-red-500"
              title={t('clip.delete')}
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
