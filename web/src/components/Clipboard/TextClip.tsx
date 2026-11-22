/**
 * TextClip — displays text content with URL linkification and expand/collapse for long text.
 */

import { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { Clip } from '../../types';
import { useClipboard } from '../../hooks/useClipboard';
import { linkifyText } from '../../utils/format';
import HighlightedText from '../Common/HighlightedText';

const MAX_LINES = 6;

interface TextClipProps {
  clip: Clip;
  onShowToast: (message: string, type: 'success' | 'error') => void;
  /** Search terms to highlight (only set while rendering search results). */
  highlightTerms?: string[];
}

export default function TextClip({ clip, onShowToast, highlightTerms }: TextClipProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const { copied, copyToClipboard } = useClipboard();

  const lineCount = clip.content.split('\n').length;
  const isLong = lineCount > MAX_LINES;

  const parts = useMemo(() => linkifyText(clip.content), [clip.content]);

  const handleCopy = async () => {
    const success = await copyToClipboard(clip.content);
    if (success) {
      onShowToast(t('clip.copied'), 'success');
    }
  };

  return (
    <div>
      {/* Text content */}
      <div
        className={`text-sm whitespace-pre-wrap break-words ${
          !expanded && isLong ? 'line-clamp-6' : ''
        }`}
      >
        {parts.map((part, i) =>
          part.type === 'link' ? (
            <a
              key={i}
              href={part.value}
              target="_blank"
              rel="noopener noreferrer"
              className="text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300 underline underline-offset-2 break-all"
            >
              {part.value}
            </a>
          ) : (
            <HighlightedText key={i} text={part.value} terms={highlightTerms} />
          ),
        )}
      </div>

      {/* Actions row */}
      <div className="flex items-center gap-2 mt-2">
        {/* Copy button */}
        <button
          onClick={handleCopy}
          className="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-600 dark:text-gray-300"
        >
          {copied ? (
            <>
              <svg className="w-3.5 h-3.5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
              </svg>
              {t('clip.copied')}
            </>
          ) : (
            <>
              <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
              </svg>
              {t('clip.copy')}
            </>
          )}
        </button>

        {/* Expand/collapse toggle */}
        {isLong && (
          <button
            onClick={() => setExpanded(!expanded)}
            className="text-xs text-blue-500 hover:text-blue-600 dark:text-blue-400 dark:hover:text-blue-300"
          >
            {expanded ? t('clip.collapse') : t('clip.expand')}
          </button>
        )}
      </div>
    </div>
  );
}
