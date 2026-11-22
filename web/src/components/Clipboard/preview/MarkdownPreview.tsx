import { useEffect, useState } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import PreviewStatus from './PreviewStatus';

export default function MarkdownPreview({ url, filename }: { url: string; filename: string }) {
  const [content, setContent] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    fetch(url, { signal: controller.signal }).then((response) => {
      if (!response.ok) throw new Error(`Preview HTTP ${response.status}`);
      return response.text();
    }).then(setContent).catch(() => {
      if (!controller.signal.aborted) setFailed(true);
    });
    return () => controller.abort();
  }, [url]);

  if (failed) return <PreviewStatus message="preview.unavailable" />;
  if (content === null) return <PreviewStatus message="preview.loading" />;

  return (
    <div role="region" aria-label={filename} tabIndex={0}
      className="max-h-[min(26rem,55dvh)] overflow-y-auto focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500">
      <article className="markdown-preview mx-auto max-w-[68ch] px-4 py-4 text-sm leading-6 text-gray-800 dark:text-gray-200">
        <Markdown remarkPlugins={[remarkGfm]} skipHtml components={{
          a: ({ node: _node, ...props }) => <a {...props} target="_blank" rel="noreferrer" />,
          img: ({ alt }) => <span>{alt}</span>,
          table: ({ node: _node, ...props }) => <div className="overflow-x-auto"><table {...props} /></div>,
        }}>
          {content}
        </Markdown>
      </article>
    </div>
  );
}
