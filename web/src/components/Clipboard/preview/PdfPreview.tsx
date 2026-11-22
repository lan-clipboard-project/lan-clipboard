import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AnnotationMode, getDocument, GlobalWorkerOptions, ResponseException } from 'pdfjs-dist/legacy/build/pdf.mjs';
import type { PDFDocumentProxy, PDFPageProxy, RenderTask } from 'pdfjs-dist';
import workerUrl from 'pdfjs-dist/legacy/build/pdf.worker.min.mjs?url';
import PreviewStatus from './PreviewStatus';

GlobalWorkerOptions.workerSrc = workerUrl;

const pageButton = 'rounded px-2 py-1 hover:bg-gray-200 disabled:opacity-40 dark:hover:bg-gray-700';

function slideStep(viewport: HTMLDivElement) {
  const [first, second] = viewport.children;
  return second ? second.getBoundingClientRect().left - first.getBoundingClientRect().left : 0;
}

function drawPage(canvas: HTMLCanvasElement, pdfPage: PDFPageProxy) {
  const { width, height } = pdfPage.getViewport({ scale: 1 });
  const viewport = pdfPage.getViewport({ scale: 1280 / Math.max(width, height) });
  canvas.width = Math.ceil(viewport.width);
  canvas.height = Math.ceil(viewport.height);
  return pdfPage.render({ canvas, viewport, annotationMode: AnnotationMode.DISABLE });
}

interface PdfPreviewProps {
  url: string;
  filename: string;
  slides: boolean;
}

interface PdfPagesProps extends Pick<PdfPreviewProps, 'filename' | 'slides'> {
  pdf: PDFDocumentProxy;
  onError: () => void;
}

export default function PdfPreview({ url, filename, slides }: PdfPreviewProps) {
  const [pdf, setPdf] = useState<PDFDocumentProxy | null>(null);
  const [error, setError] = useState('');
  const [attempt, setAttempt] = useState(0);
  const handlePageError = useCallback(() => setError('preview.unavailable'), []);

  useEffect(() => {
    let active = true;
    const loading = getDocument({ url, disableAutoFetch: true, disableStream: true, useWasm: false });
    loading.promise.then((pdf) => {
      if (active) setPdf(pdf);
    }).catch((reason) => {
      if (!active) return;
      if (reason instanceof ResponseException && reason.status === 503) {
        setError('preview.officeUnavailable');
      } else {
        setError('preview.unavailable');
      }
    });
    return () => {
      active = false;
      void loading.destroy();
    };
  }, [url, attempt]);

  if (error) {
    return <PreviewStatus message={error} onRetry={() => {
      setError('');
      setPdf(null);
      setAttempt((current) => current + 1);
    }} />;
  }

  if (!pdf) return <PreviewStatus message="preview.loading" />;
  return <PdfPages pdf={pdf} filename={filename} slides={slides} onError={handlePageError} />;
}

function PdfPages({ pdf, filename, slides, onError }: PdfPagesProps) {
  const { t } = useTranslation();
  const viewportRef = useRef<HTMLDivElement>(null);
  const [firstRatio, setFirstRatio] = useState(0);
  const [visiblePages, setVisiblePages] = useState(() => new Set<number>());
  const [currentSlide, setCurrentSlide] = useState(1);
  const [hasOverflow, setHasOverflow] = useState(false);
  const pageCount = pdf.numPages;

  useEffect(() => {
    let active = true;
    pdf.getPage(1).then((page) => {
      const { width, height } = page.getViewport({ scale: 1 });
      if (active) setFirstRatio(width / height);
    }).catch(() => {
      if (active) onError();
    });
    return () => { active = false; };
  }, [pdf, onError]);

  useEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    if (typeof IntersectionObserver === 'undefined') {
      setVisiblePages(new Set(Array.from({ length: pageCount }, (_, i) => i + 1)));
      return;
    }
    const observer = new IntersectionObserver((entries) => {
      setVisiblePages((previous) => {
        const next = new Set(previous);
        for (const entry of entries) {
          const page = Number((entry.target as HTMLElement).dataset.page);
          if (entry.isIntersecting) next.add(page);
          else next.delete(page);
        }
        return next;
      });
    });
    for (const page of viewport.children) observer.observe(page);
    return () => observer.disconnect();
  }, [firstRatio, pageCount]);

  const updateSlidePosition = useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    const maxScroll = viewport.scrollWidth - viewport.clientWidth;
    const step = slideStep(viewport);
    setHasOverflow(step > 0 && maxScroll > 1);
    if (!step || viewport.scrollLeft <= 1) {
      setCurrentSlide(1);
    } else if (viewport.scrollLeft >= maxScroll - 1) {
      setCurrentSlide(pageCount);
    } else {
      setCurrentSlide(Math.round(viewport.scrollLeft / step) + 1);
    }
  }, [pageCount]);

  useEffect(() => {
    const viewport = viewportRef.current;
    if (!slides || !viewport) return;
    const observer = new ResizeObserver(updateSlidePosition);
    observer.observe(viewport);
    return () => observer.disconnect();
  }, [slides, firstRatio, updateSlidePosition]);

  function stepSlides(direction: -1 | 1) {
    const viewport = viewportRef.current;
    if (!viewport) return;
    const step = slideStep(viewport);
    // At the end, the last slide snaps to the right edge. Find the next
    // slide boundary from the actual scroll position, even at that offset.
    const index = direction === -1
      ? Math.ceil((viewport.scrollLeft - 1) / step) - 1
      : Math.floor((viewport.scrollLeft + 1) / step) + 1;
    const target = viewport.children[index] as HTMLElement | undefined;
    if (!target) return;
    viewport.scrollTo({
      left: target.offsetLeft,
      behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth',
    });
  }

  if (!firstRatio) return <PreviewStatus message="preview.loading" />;

  return (
    <>
      <div ref={viewportRef} role="region" aria-label={filename} tabIndex={0}
        className={`focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-blue-500 ${
          slides ? 'relative flex snap-x snap-mandatory items-start gap-2 overflow-x-auto overflow-y-hidden overscroll-x-contain'
            : 'max-h-[min(26rem,55dvh)] space-y-2 overflow-y-auto'
        }`}
        onScroll={slides ? updateSlidePosition : undefined}>
        {Array.from({ length: pageCount }, (_, i) => {
          const page = i + 1;
          const label = t(slides ? 'preview.slide' : 'preview.page', { current: page, total: pageCount });
          // Buffer the preceding and following page and release canvases outside that window.
          const inRange = visiblePages.has(page) || visiblePages.has(page - 1) || visiblePages.has(page + 1);
          return <PdfPage key={page} pdf={pdf} page={page} label={label} slides={slides}
            initialRatio={firstRatio} inRange={inRange} onError={onError} />;
        })}
      </div>
      {slides && hasOverflow && (
        <div className="flex items-center gap-2 px-2 py-1.5 text-xs text-gray-600 dark:text-gray-300">
          <button type="button" className={pageButton} aria-label={t('preview.previousSlide')}
            disabled={currentSlide <= 1} onClick={() => stepSlides(-1)}>‹</button>
          <span aria-live="polite">{t('preview.slide', { current: currentSlide, total: pageCount })}</span>
          <button type="button" className={pageButton} aria-label={t('preview.nextSlide')}
            disabled={currentSlide >= pageCount} onClick={() => stepSlides(1)}>›</button>
        </div>
      )}
    </>
  );
}

function PdfPage({ pdf, page, label, slides, initialRatio, inRange, onError }: {
  pdf: PDFDocumentProxy;
  page: number;
  label: string;
  slides: boolean;
  initialRatio: number;
  inRange: boolean;
  onError: () => void;
}) {
  const [ratio, setRatio] = useState(initialRatio);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!inRange || !canvas) return;
    let active = true;
    let pdfPage: PDFPageProxy | undefined;
    let render: RenderTask | undefined;
    pdf.getPage(page).then((loadedPage) => {
      if (!active) return;
      pdfPage = loadedPage;
      if (!slides) {
        const { width, height } = pdfPage.getViewport({ scale: 1 });
        setRatio(width / height);
      }
      render = drawPage(canvas, pdfPage);
      return render.promise;
    }).catch(() => {
      if (active) onError();
    });
    return () => {
      active = false;
      render?.cancel();
      pdfPage?.cleanup();
    };
  }, [inRange, pdf, page, slides, onError]);

  return (
    <div data-page={page}
      style={{ aspectRatio: ratio, maxWidth: slides ? `calc(min(22rem, 45dvh) * ${ratio})` : undefined }}
      className={`relative bg-white ${
        slides ? 'w-[82%] shrink-0 snap-start last:snap-end only:w-full' : 'w-full'
      }`}>
      {inRange && (
        <canvas ref={canvasRef} role="img" aria-label={label}
          className="absolute inset-0 block h-full w-full object-contain" />
      )}
    </div>
  );
}
