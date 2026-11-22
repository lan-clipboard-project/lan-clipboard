import { useId, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

interface TimelineDateNavProps {
  dates: { key: string; label: string }[];
  activeDate: string | null;
  pendingDate: string | null;
  onSelect: (date: string) => void;
}

export default function TimelineDateNav({ dates, activeDate, pendingDate, onSelect }: TimelineDateNavProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const toggleRef = useRef<HTMLButtonElement>(null);
  const panelId = useId();

  return (
    <div className="absolute left-2 top-4 z-10 min-[1120px]:left-auto min-[1120px]:right-[calc(50%+25rem)]"
      onKeyDown={(event) => {
        if (event.key === 'Escape' && open) {
          setOpen(false);
          toggleRef.current?.focus();
        }
      }}>
      <button ref={toggleRef} type="button" aria-label={t('date.navigation')} aria-controls={panelId}
        aria-expanded={open} onClick={() => setOpen(!open)}
        className="grid h-8 w-8 place-items-center rounded-lg border border-gray-200 bg-white text-gray-600 shadow-sm hover:bg-gray-50 min-[1120px]:hidden dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300">
        <svg aria-hidden="true" viewBox="0 0 24 24" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth={1.8}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M8 3v4m8-4v4M4 10h16M6 5h12a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2z" />
        </svg>
      </button>
      <nav id={panelId} aria-label={t('date.navigation')}
        className={`${open ? 'block' : 'hidden'} absolute left-0 mt-2 w-40 rounded-xl border border-gray-200 bg-white p-1.5 shadow-sm min-[1120px]:static min-[1120px]:mt-0 min-[1120px]:block min-[1120px]:w-36 dark:border-gray-700 dark:bg-gray-800`}>
        <h2 className="px-2 py-1.5 text-xs font-semibold text-gray-500 dark:text-gray-400">{t('date.navigation')}</h2>
        <ul className="max-h-[min(24rem,55dvh)] space-y-0.5 overflow-y-auto overscroll-contain">
          {dates.map(({ key, label }) => (
            <li key={key}>
              <button type="button" aria-current={key === activeDate ? 'date' : undefined}
                aria-busy={key === pendingDate || undefined} disabled={pendingDate !== null}
                title={key} onClick={() => {
                  onSelect(key);
                  setOpen(false);
                }}
                className={`block w-full rounded-md px-2 py-1.5 text-left text-sm leading-5 ${
                  key === activeDate
                    ? 'bg-blue-50 font-semibold text-blue-700 dark:bg-blue-900/35 dark:text-blue-300'
                    : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700'
                }`}>
                <time dateTime={key}>{label}</time>
              </button>
            </li>
          ))}
        </ul>
      </nav>
    </div>
  );
}
