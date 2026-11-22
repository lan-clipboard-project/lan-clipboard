import { useTranslation } from 'react-i18next';

export default function PreviewStatus({ message, onRetry }: { message: string; onRetry?: () => void }) {
  const { t } = useTranslation();

  return (
    <div role="status" className="flex min-h-14 items-center justify-center gap-3 px-3 py-2 text-sm text-gray-500 dark:text-gray-400">
      {t(message)}
      {onRetry && (
        <button type="button" onClick={onRetry} className="text-blue-600 hover:underline dark:text-blue-400">
          {t('preview.retry')}
        </button>
      )}
    </div>
  );
}
