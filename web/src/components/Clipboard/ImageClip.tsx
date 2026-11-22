/**
 * ImageClip — displays image with thumbnail preview and download button.
 */

import { useTranslation } from 'react-i18next';
import { downloadClip } from '../../api/clips';
import type { Clip } from '../../types';
import { formatFileSize } from '../../utils/format';
import HighlightedText from '../Common/HighlightedText';

interface ImageClipProps {
  clip: Clip;
  onPreviewImage: (url: string) => void;
  /** Search terms to highlight (only set while rendering search results). */
  highlightTerms?: string[];
}

export default function ImageClip({ clip, onPreviewImage, highlightTerms }: ImageClipProps) {
  const { t } = useTranslation();
  const imgSrc = clip.hasThumb && clip.thumbUrl ? clip.thumbUrl : clip.fileUrl;

  return (
    <div>
      {/* Image preview */}
      <div
        className="cursor-pointer rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-700"
        onClick={() => onPreviewImage(clip.fileUrl)}
      >
        <img
          src={imgSrc}
          alt={clip.fileName}
          className="max-h-64 w-auto object-contain mx-auto"
          loading="lazy"
        />
      </div>

      {/* File info + download */}
      <div className="flex items-center justify-between mt-2">
        <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400 min-w-0">
          <span className="truncate">
            <HighlightedText text={clip.fileName} terms={highlightTerms} />
          </span>
          <span className="flex-shrink-0">{formatFileSize(clip.fileSize)}</span>
        </div>
        <button
          type="button"
          onClick={() => downloadClip(clip)}
          className="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-600 dark:text-gray-300 flex-shrink-0"
        >
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
          </svg>
          {t('clip.download')}
        </button>
      </div>
    </div>
  );
}
