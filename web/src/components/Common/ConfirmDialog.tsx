/**
 * ConfirmDialog — centered modal confirmation prompt.
 * Cancels on backdrop click, Escape, or the cancel button; locks body scroll while mounted.
 */

import { useTranslation } from 'react-i18next';
import Modal from './Modal';

interface ConfirmDialogProps {
  message: string;
  onConfirm: () => void;
  onClose: () => void;
  title?: string;
  open?: boolean;
}

export default function ConfirmDialog({ message, title, open, onConfirm, onClose }: ConfirmDialogProps) {
  const { t } = useTranslation();

  return (
    <Modal label={title ?? message} backdropClassName="bg-black/50" open={open} onClose={onClose}>
      {(close) => (
        <div
          data-modal-panel
          className="w-full max-w-sm p-5 rounded-xl bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 shadow-xl motion-safe:animate-fade-in-up"
        >
          {title && (
            <h2 className="mb-2 text-base font-semibold text-gray-900 dark:text-gray-100">{title}</h2>
          )}

          <p className="text-sm text-gray-600 dark:text-gray-300">{message}</p>

          <div className="flex justify-end gap-2 mt-5">
            <button
              type="button"
              onClick={() => close()}
              className="px-4 py-2 text-sm rounded-lg bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200"
            >
              {t('dialog.cancel')}
            </button>
            <button
              type="button"
              onClick={() => close(onConfirm)}
              className="px-4 py-2 text-sm rounded-lg bg-red-600 hover:bg-red-700 active:bg-red-800 text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-red-500"
            >
              {t('dialog.confirm')}
            </button>
          </div>
        </div>
      )}
    </Modal>
  );
}
