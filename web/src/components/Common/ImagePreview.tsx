/**
 * ImagePreview — full-screen image lightbox.
 * Closes on backdrop click, close button, or Escape; locks body scroll while mounted.
 */

import { useTranslation } from 'react-i18next';
import Modal from './Modal';

interface ImagePreviewProps {
  url: string;
  onClose: () => void;
}

export default function ImagePreview({ url, onClose }: ImagePreviewProps) {
  const { t } = useTranslation();

  return (
    <Modal label={t('clip.preview')} backdropClassName="bg-black/80" onClose={onClose}>
      {(close) => (
        <>
          <img
            data-modal-panel
            src={url}
            alt={t('clip.preview')}
            className="max-h-[90vh] max-w-[90vw] object-contain rounded-lg shadow-2xl motion-safe:animate-fade-in"
          />

          <button
            type="button"
            onClick={() => close()}
            className="absolute top-4 right-4 p-2 rounded-full bg-black/50 hover:bg-black/70 text-white"
            title={t('dialog.close')}
            aria-label={t('dialog.close')}
          >
            <svg
              className="w-6 h-6"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </>
      )}
    </Modal>
  );
}
