/**
 * UploadProgress — displays upload progress bars for files being uploaded.
 */

import type { UploadFileProgress } from '../../types';

interface UploadProgressProps {
  uploads: UploadFileProgress[];
}

export default function UploadProgress({ uploads }: UploadProgressProps) {
  if (uploads.length === 0) return null;

  return (
    <div className="space-y-2">
      {uploads.map((item, i) => (
        <div key={i} className="flex items-center gap-3 text-xs">
          <span className="truncate max-w-[200px] text-gray-600 dark:text-gray-400">
            {item.file.name}
          </span>
          <div className="flex-1 h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-300 ${
                item.status === 'error'
                  ? 'bg-red-500'
                  : item.status === 'done'
                    ? 'bg-green-500'
                    : 'bg-blue-500'
              }`}
              style={{ width: `${item.progress}%` }}
            />
          </div>
          <span className="text-gray-500 dark:text-gray-400 flex-shrink-0 w-10 text-right">
            {item.status === 'error' ? '✗' : `${item.progress}%`}
          </span>
        </div>
      ))}
    </div>
  );
}
