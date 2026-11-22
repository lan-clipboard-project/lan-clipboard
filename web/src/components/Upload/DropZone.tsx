/**
 * DropZone — full-page overlay shown when files are dragged over the window.
 */

import { useTranslation } from 'react-i18next';

export default function DropZone() {
  const { t } = useTranslation();

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-blue-500/20 dark:bg-blue-500/10 backdrop-blur-sm pointer-events-none animate-fade-in">
      <div className="flex flex-col items-center gap-4 p-12 rounded-2xl border-2 border-dashed border-blue-400 dark:border-blue-500 bg-white/80 dark:bg-gray-800/80">
        <svg className="w-16 h-16 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
        </svg>
        <p className="text-lg font-medium text-blue-600 dark:text-blue-400">
          {t('dropzone.hint')}
        </p>
      </div>
    </div>
  );
}
