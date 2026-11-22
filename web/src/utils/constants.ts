/** Application-wide constants */

/** Maximum text content size in bytes (300KB) */
export const MAX_TEXT_SIZE = 300 * 1024;

/** Maximum file size in bytes (300MB) */
export const MAX_FILE_SIZE = 300 * 1024 * 1024;

/** Number of clips to fetch per page */
export const PAGE_SIZE = 20;

/** Debounce delay for search input in milliseconds */
export const SEARCH_DEBOUNCE_MS = 300;

/** Toast auto-dismiss duration in milliseconds */
export const TOAST_DURATION_MS = 2000;

/** Infinite scroll trigger threshold in pixels from bottom */
export const SCROLL_THRESHOLD = 100;

/** WebSocket reconnect settings */
export const WS_INITIAL_DELAY_MS = 1000;
export const WS_MAX_DELAY_MS = 30000;
export const WS_BACKOFF_FACTOR = 2;

/** LocalStorage keys */
export const LS_THEME_KEY = 'theme';
export const LS_LANG_KEY = 'lang';
