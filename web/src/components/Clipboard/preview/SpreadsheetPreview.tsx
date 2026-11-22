import { useEffect, useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { SheetRequest, SheetResponse } from './spreadsheet.worker';
import PreviewStatus from './PreviewStatus';

export default function SpreadsheetPreview({ url }: { url: string }) {
  const { t } = useTranslation();
  const sheetSelectId = useId();
  const workerRef = useRef<Worker | null>(null);
  const [preview, setPreview] = useState<SheetResponse | null>(null);

  useEffect(() => {
    const worker = new Worker(new URL('./spreadsheet.worker.ts', import.meta.url), { type: 'module' });
    workerRef.current = worker;
    worker.onmessage = ({ data }: MessageEvent<SheetResponse>) => setPreview(data);
    worker.onerror = () => setPreview({ failed: true });
    worker.postMessage({ url } satisfies SheetRequest);
    return () => {
      workerRef.current = null;
      worker.terminate();
    };
  }, [url]);

  if (!preview) return <PreviewStatus message="preview.loading" />;
  if ('failed' in preview) return <PreviewStatus message="preview.unavailable" />;

  const { columns, rows, name, limit } = preview.grid;
  return (
    <div className="text-xs text-gray-700 dark:text-gray-200">
      {preview.sheets.length > 1 && (
        <div className="flex items-center gap-2 px-2 py-1.5">
          <label htmlFor={sheetSelectId} className="shrink-0 font-medium">{t('preview.sheet')}</label>
          <select id={sheetSelectId} value={name}
            onChange={(event) => workerRef.current?.postMessage({ sheet: event.target.value } satisfies SheetRequest)}
            className="min-w-0 rounded border border-gray-200 bg-white px-2 py-1 dark:border-gray-700 dark:bg-gray-800">
            {preview.sheets.map((sheet) => <option key={sheet}>{sheet}</option>)}
          </select>
        </div>
      )}
      {rows.length ? (
        <div tabIndex={0} role="region" aria-label={name} className="max-h-64 overflow-auto">
          <table aria-label={name} className="w-max min-w-full border-separate border-spacing-0 text-left">
            <thead className="sticky top-0 z-10 bg-gray-100 dark:bg-gray-800">
              <tr>
                <th />
                {columns.map((column) => <th key={column} scope="col" className="px-2 py-1 font-medium">{column}</th>)}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={index} className="even:bg-gray-100/60 dark:even:bg-gray-800/60">
                  <th scope="row" className="bg-gray-100 px-2 py-1 text-right font-medium dark:bg-gray-800">{index + 1}</th>
                  {columns.map((column, colIndex) => (
                    <td key={column} className="min-w-16 max-w-xs whitespace-pre-wrap break-words border-b border-r border-gray-200 px-2 py-1 dark:border-gray-700">
                      {row[colIndex]}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : <PreviewStatus message="preview.emptySheet" />}
      {limit && <p className="px-2 py-1.5 text-gray-500 dark:text-gray-400">{t('preview.sheetLimit', limit)}</p>}
    </div>
  );
}
