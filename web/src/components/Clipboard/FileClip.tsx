/**
 * FileClip — displays the file preview, details, and actions.
 */

import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { downloadClip } from '../../api/clips';
import type { Clip } from '../../types';
import { formatFileSize, getFileIcon } from '../../utils/format';
import HighlightedText from '../Common/HighlightedText';
import FilePreview, { getDocumentPreview, getFilePreviewType } from './preview/FilePreview';

const actionClass = 'inline-flex items-center gap-1 whitespace-nowrap rounded-lg px-3 py-1.5 text-xs';

interface FileClipProps {
  clip: Clip;
  /** Search terms to highlight (only set while rendering search results). */
  highlightTerms?: string[];
}

export default function FileClip({ clip, highlightTerms }: FileClipProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const previewId = useId();
  const iconUrl = getFileIcon(clip.fileName);
  const previewType = getFilePreviewType(clip);
  const documentPreview = getDocumentPreview(clip, previewType);
  const openUrl = documentPreview?.canOpen ? documentPreview.url : null;
  const previewAction = t(expanded ? 'preview.collapse' : 'preview.expand');

  return (
    <div className="space-y-3">
      <div className={`flex items-center gap-x-3 gap-y-2 ${openUrl ? 'flex-wrap' : ''}`}>
        <div className={`flex min-w-0 items-center gap-3 ${openUrl ? 'flex-[1_1_10rem]' : 'flex-1'}`}>
          <div className="flex shrink-0 items-center gap-1">
            {previewType && (
              <button type="button" aria-label={previewAction} title={previewAction}
                aria-expanded={expanded} aria-controls={previewId}
                onClick={() => setExpanded((current) => !current)}
                className="grid h-8 w-8 place-items-center rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-200">
                <svg aria-hidden="true" viewBox="0 0 16 16" fill="currentColor"
                  className={`h-4 w-4 motion-safe:transition-transform ${expanded ? 'rotate-90' : ''}`}>
                  <path d="M5.5 3.5 11.5 8l-6 4.5z" />
                </svg>
              </button>
            )}
            <img src={iconUrl} alt="" width={32} height={32} className="h-8 w-8 shrink-0 object-contain" />
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">
              <HighlightedText text={clip.fileName} terms={highlightTerms} />
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400">{formatFileSize(clip.fileSize)}</p>
          </div>
        </div>

        <div className="ml-auto flex max-w-full shrink-0 flex-wrap justify-end gap-2">
          {openUrl && (
            <button type="button" onClick={() => window.open(openUrl, '_blank', 'noopener,noreferrer')}
              className={`${actionClass} bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-700/40 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-200`}>
              <svg aria-hidden="true" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M13 5h6v6m0-6-9 9m0-8H7a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h9a2 2 0 0 0 2-2v-3" />
              </svg>
              {t('clip.openInNewWindow')}
            </button>
          )}
          <button type="button" onClick={() => downloadClip(clip)}
            className={`${actionClass} bg-blue-50 text-blue-600 hover:bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 dark:hover:bg-blue-900/50`}>
            <svg aria-hidden="true" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            {t('clip.download')}
          </button>
        </div>
      </div>
      {previewType && (
        <FilePreview id={previewId} expanded={expanded} clip={clip} type={previewType} documentPreview={documentPreview} />
      )}
    </div>
  );
}
