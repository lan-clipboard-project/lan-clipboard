/** Formatting utility functions */

/**
 * Format a byte count into a human-readable string (e.g., "1.5 MB").
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const size = bytes / Math.pow(k, i);
  return `${size % 1 === 0 ? size : size.toFixed(1)} ${units[i]}`;
}

/**
 * Format a date string into a human-readable relative label.
 * Returns 'today', 'yesterday', or the localized date string.
 */
export function getDateLabel(dateKey: string, t: (key: string) => string, locale?: string): string {
  const now = new Date();
  const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1);

  if (dateKey === getDateKey(now.toISOString())) return t('date.today');
  if (dateKey === getDateKey(yesterday.toISOString())) return t('date.yesterday');

  const date = new Date(`${dateKey}T00:00:00`);
  return date.toLocaleDateString(locale, {
    year: date.getFullYear() === now.getFullYear() ? undefined : 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

/** Returns the exclusive upper boundary of a local calendar day in UTC. */
export function getNextDayCursor(dateKey: string): string {
  const date = new Date(`${dateKey}T00:00:00`);
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1).toISOString();
}

/**
 * Format a date string into a time string (HH:MM).
 */
export function formatTime(dateStr: string, locale?: string): string {
  const date = new Date(dateStr);
  return date.toLocaleTimeString(locale, {
    hour: '2-digit',
    minute: '2-digit',
  });
}

/**
 * Get the date key (YYYY-MM-DD) for grouping clips by date.
 */
export function getDateKey(dateStr: string): string {
  const date = new Date(dateStr);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`;
}

const fileIconExtensions = new Set([
  '7z', 'avi', 'bz2', 'css', 'csv', 'doc', 'docx', 'flac', 'gif', 'go', 'gz',
  'heic', 'html', 'java', 'jpeg', 'jpg', 'js', 'json', 'md', 'mkv', 'mov', 'mp3',
  'mp4', 'ods', 'odt', 'pdf', 'png', 'ppt', 'pptx', 'py', 'rar', 'svg', 'tar',
  'ts', 'txt', 'wav', 'webp', 'xls', 'xlsm', 'xlsx', 'xml', 'xz', 'zip',
]);

/** Get the URL for a locally bundled file type icon. */
export function getFileIcon(fileName: string): string {
  const dot = fileName.lastIndexOf('.');
  const extension = dot > 0 ? fileName.slice(dot + 1).toLowerCase() : '';
  const icon = fileIconExtensions.has(extension) ? extension : 'blank';
  return `${import.meta.env.BASE_URL}assets/file-icons/${icon}.svg`;
}

/**
 * Auto-detect URLs in text and wrap them in anchor tags.
 * Returns an array of string and JSX element segments.
 */
export function linkifyText(text: string): Array<{ type: 'text' | 'link'; value: string }> {
  const urlPattern = /https?:\/\/[^\s<>"{}|\\^`[\]]+/g;
  const parts: Array<{ type: 'text' | 'link'; value: string }> = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = urlPattern.exec(text)) !== null) {
    if (match.index > lastIndex) {
      parts.push({ type: 'text', value: text.slice(lastIndex, match.index) });
    }
    parts.push({ type: 'link', value: match[0] });
    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < text.length) {
    parts.push({ type: 'text', value: text.slice(lastIndex) });
  }

  if (parts.length === 0) {
    parts.push({ type: 'text', value: text });
  }

  return parts;
}
