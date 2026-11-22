/**
 * HighlightedText — wraps search-term occurrences in a themed <mark>.
 * Only used while rendering search results; without terms it renders plain text.
 */

import { useMemo } from 'react';

interface HighlightedTextProps {
  text: string;
  terms?: string[];
}

/** Escape characters that have a special meaning inside a RegExp. */
function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

export default function HighlightedText({ text, terms }: HighlightedTextProps) {
  const regex = useMemo(() => {
    const cleaned = Array.from(
      new Set(
        (terms ?? [])
          .map((term) => term.trim())
          .filter((term) => term.length > 0),
      ),
    ).sort((a, b) => b.length - a.length);

    if (cleaned.length === 0) return null;
    return new RegExp(`(${cleaned.map(escapeRegExp).join('|')})`, 'gi');
  }, [terms]);

  if (!regex) return <>{text}</>;

  // split() keeps the captured matches as the odd-indexed parts.
  const parts = text.split(regex);
  return (
    <>
      {parts.map((part, index) =>
        index % 2 === 1 ? (
          <mark
            key={index}
            className="rounded-sm bg-blue-100 px-0.5 font-semibold text-blue-700 dark:bg-blue-400/20 dark:text-blue-300"
          >
            {part}
          </mark>
        ) : (
          <span key={index}>{part}</span>
        ),
      )}
    </>
  );
}
