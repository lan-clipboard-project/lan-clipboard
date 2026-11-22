/** Core type definitions for the LAN Clipboard application */

export type ClipType = 'text' | 'image' | 'file';

/** Represents a single clipboard item */
export interface Clip {
  id: string;
  type: ClipType;
  content: string;
  fileName: string;
  fileSize: number;
  mimeType: string;
  hasThumb: boolean;
  fileUrl: string;
  thumbUrl: string;
  pinned: boolean;
  createdAt: string;
  uploader?: string;
}

/** WebSocket message from the server */
export interface WSMessage {
  type: 'clip_created' | 'clip_deleted' | 'clip_updated' | 'clips_cleared';
  data: Clip | { id: string };
}

/** Connection status for the WebSocket */
export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected';

/** Theme mode options */
export type ThemeMode = 'light' | 'dark' | 'system';

/** Paginated clip list response from GET /api/clips */
export interface ClipPage {
  clips: Clip[];
  hasMore: boolean;
  /** Opaque cursor for the next page ("" when there is no next page) */
  nextCursor: string;
}

/** Upload progress tracking per file */
export interface UploadFileProgress {
  file: File;
  progress: number;
  status: 'pending' | 'uploading' | 'done' | 'error';
}
