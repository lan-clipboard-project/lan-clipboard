/**
 * REST API client for clipboard operations.
 * All endpoints are relative to the same origin.
 */

import type { Clip, ClipPage, ClipType } from '../types';

/** Fetch paginated clips with optional type filter */
export async function fetchClips(
  cursor?: string,
  limit: number = 20,
  type?: ClipType,
  direction: 'older' | 'newer' = 'older',
): Promise<ClipPage> {
  const params = new URLSearchParams();
  if (cursor) params.set(direction === 'newer' ? 'after' : 'cursor', cursor);
  params.set('limit', String(limit));
  if (type) params.set('type', type);

  const res = await fetch(`/api/clips?${params.toString()}`);
  if (!res.ok) throw new Error(`Failed to fetch clips: ${res.status}`);
  return res.json();
}

export async function fetchClipDates(type: ClipType | null, signal: AbortSignal): Promise<string[]> {
  const params = new URLSearchParams({ timezone: Intl.DateTimeFormat().resolvedOptions().timeZone });
  if (type) params.set('type', type);
  const res = await fetch(`/api/clips/dates?${params}`, { signal });
  if (!res.ok) throw new Error(`Failed to fetch clip dates: ${res.status}`);
  const data: { dates: string[] } = await res.json();
  return data.dates;
}

/** Fetch all pinned clips */
export async function fetchPinnedClips(): Promise<Clip[]> {
  const res = await fetch('/api/clips/pinned');
  if (!res.ok) throw new Error(`Failed to fetch pinned clips: ${res.status}`);
  const data: { clips: Clip[] } = await res.json();
  return data.clips ?? [];
}

/** Create a new text clip */
export async function createTextClip(content: string): Promise<Clip> {
  const res = await fetch('/api/clips', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content }),
  });
  if (!res.ok) throw new Error(`Failed to create clip: ${res.status}`);
  return res.json();
}

/**
 * Upload files individually with progress tracking via XMLHttpRequest.
 * Returns an array of created Clip objects.
 */
export async function uploadFiles(
  files: File[],
  onProgress?: (file: File, progress: number) => void
): Promise<Clip[]> {
  const results: Clip[] = [];

  for (const file of files) {
    const clips = await new Promise<Clip[]>((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      const formData = new FormData();
      formData.append('file', file);

      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable && onProgress) {
          onProgress(file, Math.round((e.loaded / e.total) * 100));
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            // The server responds with { clips: [...] } for uploads.
            const data: { clips?: Clip[] } | Clip[] = JSON.parse(xhr.responseText);
            resolve(Array.isArray(data) ? data : (data.clips ?? []));
          } catch {
            reject(new Error('Invalid response'));
          }
        } else {
          reject(new Error(`Upload failed: ${xhr.status}`));
        }
      });

      xhr.addEventListener('error', () => reject(new Error('Upload failed')));
      xhr.addEventListener('abort', () => reject(new Error('Upload aborted')));

      xhr.open('POST', '/api/clips/upload');
      xhr.send(formData);
    });

    results.push(...clips);
  }

  return results;
}

/** Delete a single clip by ID */
export async function deleteClip(id: string): Promise<void> {
  const res = await fetch(`/api/clips/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(`Failed to delete clip: ${res.status}`);
}

/** Starts a native browser download without fetching the file into JavaScript memory. */
export function downloadClip({ fileUrl, fileName }: Pick<Clip, 'fileUrl' | 'fileName'>) {
  const link = document.createElement('a');
  link.href = fileUrl;
  link.download = fileName;
  link.hidden = true;
  document.body.append(link);
  link.click();
  link.remove();
}

/** Delete all clips */
export async function deleteAllClips(): Promise<void> {
  const res = await fetch('/api/clips', { method: 'DELETE' });
  if (!res.ok) throw new Error(`Failed to delete all clips: ${res.status}`);
}

/** Update a clip (currently supports pinned toggle) */
export async function updateClip(id: string, pinned: boolean): Promise<Clip> {
  const res = await fetch(`/api/clips/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pinned }),
  });
  if (!res.ok) throw new Error(`Failed to update clip: ${res.status}`);
  return res.json();
}

/** Search clips by query and optional type filter */
export async function searchClips(query: string, type?: ClipType): Promise<Clip[]> {
  const params = new URLSearchParams({ q: query });
  if (type) params.set('type', type);

  const res = await fetch(`/api/clips/search?${params.toString()}`);
  if (!res.ok) throw new Error(`Failed to search clips: ${res.status}`);
  const data: { clips: Clip[] } = await res.json();
  return data.clips ?? [];
}
