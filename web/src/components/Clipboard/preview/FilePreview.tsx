import { lazy, Suspense, useState } from 'react';
import type { Clip } from '../../../types';
import PreviewStatus from './PreviewStatus';

const MarkdownPreview = lazy(() => import('./MarkdownPreview'));
const SpreadsheetPreview = lazy(() => import('./SpreadsheetPreview'));
const PdfPreview = lazy(() => import('./PdfPreview'));

const audioExtensions = new Set(['aac', 'flac', 'm4a', 'mp3', 'oga', 'ogg', 'opus', 'wav']);
const videoExtensions = new Set(['avi', 'm4v', 'mkv', 'mov', 'mp4', 'ogv', 'webm']);
const markdownExtensions = new Set(['md', 'markdown', 'mdown', 'mkd']);
const MiB = 1024 * 1024;

export function getFilePreviewType({ fileName, fileUrl }: Clip) {
  if (!fileUrl) return null;
  const extension = /\.([^.]+)$/.exec(fileName)?.[1].toLowerCase() ?? '';
  if (audioExtensions.has(extension)) return 'audio';
  if (videoExtensions.has(extension)) return 'video';
  if (markdownExtensions.has(extension)) return 'markdown';
  if (extension === 'html' || extension === 'htm') return 'html';
  if (extension === 'xlsx') return 'spreadsheet';
  if (extension === 'docx') return 'document';
  if (extension === 'pptx') return 'slides';
  if (extension === 'pdf') return 'pdf';
  return null;
}

type FilePreviewType = NonNullable<ReturnType<typeof getFilePreviewType>>;

export function getDocumentPreview({ id, fileSize, fileUrl }: Clip, type: FilePreviewType | null) {
  if (type === 'pdf') {
    return { url: `${fileUrl}?preview=1`, slides: false, tooLarge: fileSize > 100 * MiB, canOpen: true };
  }
  if (type === 'document' || type === 'slides') {
    const tooLarge = fileSize > 50 * MiB;
    return { url: `/api/clips/${encodeURIComponent(id)}/preview`, slides: type === 'slides', tooLarge, canOpen: !tooLarge };
  }
  return null;
}

function MediaPreview({ clip, type }: { clip: Clip; type: 'audio' | 'video' }) {
  const [failed, setFailed] = useState(false);
  const source = `${clip.fileUrl}?preview=1`;

  if (failed) return <PreviewStatus message="preview.mediaUnavailable" />;
  if (type === 'audio') {
    return (
      <audio src={source} aria-label={clip.fileName} onError={() => setFailed(true)}
        controls preload="none" className="block w-full" />
    );
  }
  return (
    <video src={source} aria-label={clip.fileName} onError={() => setFailed(true)}
      controls playsInline preload="metadata" className="mx-auto block max-h-72 w-full bg-black" />
  );
}

export default function FilePreview({ id, expanded, clip, type, documentPreview }: {
  id: string;
  expanded: boolean;
  clip: Clip;
  type: FilePreviewType;
  documentPreview: ReturnType<typeof getDocumentPreview>;
}) {
  if (!expanded) return <div id={id} hidden />;

  let preview;
  if (documentPreview && !documentPreview.tooLarge) {
    preview = <PdfPreview url={documentPreview.url} filename={clip.fileName} slides={documentPreview.slides} />;
  } else if (type === 'audio' || type === 'video') {
    preview = <MediaPreview clip={clip} type={type} />;
  } else if (type === 'markdown' && clip.fileSize <= MiB) {
    preview = <MarkdownPreview url={clip.fileUrl} filename={clip.fileName} />;
  } else if (type === 'html' && clip.fileSize <= 5 * MiB) {
    preview = (
      <iframe src={`${clip.fileUrl}?preview=1`} sandbox="" referrerPolicy="no-referrer"
        title={clip.fileName} className="block h-[min(26rem,55dvh)] w-full bg-white" />
    );
  } else if (type === 'spreadsheet' && clip.fileSize <= 20 * MiB) {
    preview = <SpreadsheetPreview url={clip.fileUrl} />;
  } else {
    preview = <PreviewStatus message="preview.tooLarge" />;
  }

  return (
    <div id={id} className="min-h-14 overflow-hidden rounded-lg border border-gray-300 bg-gray-50 dark:border-gray-600 dark:bg-gray-900/60">
      <Suspense fallback={<PreviewStatus message="preview.loading" />}>
        {preview}
      </Suspense>
    </div>
  );
}
