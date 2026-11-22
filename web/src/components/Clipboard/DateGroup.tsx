/**
 * DateGroup — a date header with a list of clip cards below it.
 */

import type { ReactNode } from 'react';
import ClipPresence from './ClipPresence';

interface DateGroupProps {
  date: string;
  label: string;
  exiting: boolean;
  children: ReactNode;
}

export default function DateGroup({ date, label, exiting, children }: DateGroupProps) {
  return (
    <ClipPresence exiting={exiting}>
      <section data-timeline-date={date} aria-labelledby={`timeline-${date}`} className="pb-6">
        <h2 id={`timeline-${date}`} tabIndex={-1}
          className="mb-4 flex items-center gap-4 text-center text-xl font-bold text-gray-900 focus:outline-none sm:text-2xl dark:text-gray-100">
          <span aria-hidden="true" className="h-px flex-1 bg-gray-200 dark:bg-gray-700" />
          <time dateTime={date}>{label}</time>
          <span aria-hidden="true" className="h-px flex-1 bg-gray-200 dark:bg-gray-700" />
        </h2>
        <div className="flow-root"><div className="-mb-3">{children}</div></div>
      </section>
    </ClipPresence>
  );
}
