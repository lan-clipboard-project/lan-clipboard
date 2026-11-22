/**
 * Toast — transient success/error notification anchored above the bottom input area.
 * Auto-dismisses after TOAST_DURATION_MS; the parent clears it via onClose.
 */

import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { TOAST_DURATION_MS } from '../../utils/constants';

interface ToastProps {
  message: string;
  type: 'success' | 'error';
  onClose: () => void;
}

export default function Toast({ message, type, onClose }: ToastProps) {
  const { t } = useTranslation();
  const isSuccess = type === 'success';

  // Keep the latest onClose callback without restarting the auto-dismiss timer
  // when the parent re-renders (e.g. while upload progress updates).
  const onCloseRef = useRef(onClose);
  useEffect(() => {
    onCloseRef.current = onClose;
  }, [onClose]);

  // Auto-dismiss; clear the pending timer on unmount.
  useEffect(() => {
    const timer = setTimeout(() => onCloseRef.current(), TOAST_DURATION_MS);
    return () => clearTimeout(timer);
  }, []);

  return (
    <div className="fixed inset-x-0 bottom-24 z-40 flex justify-center px-4 pointer-events-none">
      <div
        role="status"
        aria-live="polite"
        className="pointer-events-auto flex items-center gap-2.5 max-w-md px-4 py-3 rounded-xl bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 shadow-lg animate-fade-in-up"
      >
        {isSuccess ? (
          <svg
            className="w-5 h-5 flex-shrink-0 text-green-500"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
        ) : (
          <svg
            className="w-5 h-5 flex-shrink-0 text-red-500"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
            />
          </svg>
        )}

        <span className="text-sm text-gray-800 dark:text-gray-100 break-words">{message}</span>

        <button
          type="button"
          onClick={onClose}
          className="flex-shrink-0 p-1 rounded-md text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-200"
          title={t('dialog.close')}
          aria-label={t('dialog.close')}
        >
          <svg
            className="w-4 h-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>
  );
}
