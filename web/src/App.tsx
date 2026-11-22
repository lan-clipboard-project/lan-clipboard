/**
 * Root application component.
 * Sets up the layout: Header (top), ClipList (scrollable middle), InputArea (bottom).
 * Manages WebSocket connection, initial data fetching, drag-and-drop overlay, and toast notifications.
 */

import { useEffect, useLayoutEffect, useState, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useWebSocket } from './hooks/useWebSocket';
import { useTheme } from './hooks/useTheme';
import { useClipStore } from './store/clipStore';
import Header from './components/Layout/Header';
import InputArea from './components/Layout/InputArea';
import ClipList from './components/Clipboard/ClipList';
import DropZone from './components/Upload/DropZone';
import Toast from './components/Common/Toast';
import ImagePreview from './components/Common/ImagePreview';
import { uploadFiles } from './api/clips';
import { MAX_FILE_SIZE } from './utils/constants';
import { formatFileSize } from './utils/format';
import type { UploadFileProgress } from './types';

export default function App() {
  const { t } = useTranslation();
  const { status } = useWebSocket();
  const { mode, cycleMode } = useTheme();

  const shellRef = useRef<HTMLDivElement>(null);
  const topBarRef = useRef<HTMLElement>(null);
  const bottomBarRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const shell = shellRef.current;
    const topBar = topBarRef.current;
    const bottomBar = bottomBarRef.current;
    if (!shell || !topBar || !bottomBar) return;

    // Search, the composer, and upload progress can all change the reserved space.
    const updateBarHeights = () => {
      shell.style.setProperty('--top-bar-height', `${topBar.getBoundingClientRect().height}px`);
      shell.style.setProperty('--bottom-bar-height', `${bottomBar.getBoundingClientRect().height}px`);
    };
    const observer = new ResizeObserver(updateBarHeights);
    updateBarHeights();
    observer.observe(topBar, { box: 'border-box' });
    observer.observe(bottomBar, { box: 'border-box' });
    return () => observer.disconnect();
  }, []);

  const fetchClips = useClipStore((s) => s.fetchClips);
  const fetchPinned = useClipStore((s) => s.fetchPinned);
  const addClip = useClipStore((s) => s.addClip);

  // Toast state
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);
  const showToast = useCallback((message: string, type: 'success' | 'error' = 'success') => {
    setToast({ message, type });
  }, []);

  // Image preview state
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);

  // Upload progress state
  const [uploads, setUploads] = useState<UploadFileProgress[]>([]);

  // Drag-and-drop state
  const [isDragging, setIsDragging] = useState(false);
  const dragCounter = useRef(0);

  const touchPress = useRef<{ control: HTMLElement; pointerId: number } | null>(null);

  // Initial data fetch
  useEffect(() => {
    fetchClips();
    fetchPinned();
  }, [fetchClips, fetchPinned]);

  // Re-sync state after a WebSocket reconnect: events sent while the socket
  // was down are lost, so refresh the lists once the connection is back.
  const wasConnected = useRef(false);
  useEffect(() => {
    if (status !== 'connected') return;
    if (!wasConnected.current) {
      // The initial fetch above already covers the first connection.
      wasConnected.current = true;
      return;
    }
    fetchClips();
    fetchPinned();
  }, [status, fetchClips, fetchPinned]);

  // File upload handler shared by InputArea and DropZone
  const handleUploadFiles = useCallback(
    async (files: File[]) => {
      // Validate file sizes
      const oversized = files.filter((f) => f.size > MAX_FILE_SIZE);
      if (oversized.length > 0) {
        showToast(t('input.fileTooLarge', { max: formatFileSize(MAX_FILE_SIZE) }), 'error');
        return;
      }

      // Track progress for each file
      const progressItems: UploadFileProgress[] = files.map((f) => ({
        file: f,
        progress: 0,
        status: 'pending' as const,
      }));
      setUploads(progressItems);

      try {
        const created = await uploadFiles(files, (file, progress) => {
          setUploads((prev) =>
            prev.map((u) =>
              u.file === file
                ? { ...u, progress, status: progress < 100 ? 'uploading' : 'done' }
                : u,
            ),
          );
        });
        // Add the new clips immediately so the UI updates even if the
        // WebSocket broadcast is missed (e.g. while reconnecting).
        created.forEach((clip) => addClip(clip));
        // Clear uploads after a brief delay
        setTimeout(() => setUploads([]), 1500);
      } catch {
        setUploads((prev) => prev.map((u) => ({ ...u, status: 'error' as const })));
        showToast(t('error.uploadFailed'), 'error');
        setTimeout(() => setUploads([]), 3000);
      }
    },
    [addClip, showToast, t],
  );

  // Global drag-and-drop handlers
  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounter.current++;
    if (e.dataTransfer.types.includes('Files')) {
      setIsDragging(true);
    }
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounter.current--;
    if (dragCounter.current === 0) {
      setIsDragging(false);
    }
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      dragCounter.current = 0;
      setIsDragging(false);

      const files = Array.from(e.dataTransfer.files);
      if (files.length > 0) {
        handleUploadFiles(files);
      }
    },
    [handleUploadFiles],
  );

  // Show feedback on touch down, even when the browser delays :active.
  const handleTouchPress = (event: React.PointerEvent) => {
    if (event.pointerType !== 'touch' || !event.isPrimary || !(event.target instanceof Element)) return;
    const control = event.target.closest<HTMLButtonElement>('button:enabled');
    if (!control) return;

    touchPress.current?.control.removeAttribute('data-touch-pressed');
    control.setAttribute('data-touch-pressed', '');
    touchPress.current = { control, pointerId: event.pointerId };
  };

  const clearTouchPress = (event: React.PointerEvent) => {
    const press = touchPress.current;
    if (press?.pointerId !== event.pointerId) return;
    press.control.removeAttribute('data-touch-pressed');
    touchPress.current = null;
  };

  return (
    <div
      ref={shellRef}
      className="flex flex-col h-screen overflow-hidden"
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      onPointerDown={handleTouchPress}
      onPointerUp={clearTouchPress}
      onPointerCancel={clearTouchPress}
      onLostPointerCapture={clearTouchPress}
    >
      <Header
        ref={topBarRef}
        themeMode={mode}
        onThemeToggle={cycleMode}
        onShowToast={showToast}
      />

      <ClipList onPreviewImage={setPreviewUrl} onShowToast={showToast} />

      <InputArea
        ref={bottomBarRef}
        onUploadFiles={handleUploadFiles}
        uploads={uploads}
        onShowToast={showToast}
      />

      {/* Drag-and-drop overlay */}
      {isDragging && <DropZone />}

      {/* Toast notification */}
      {toast && (
        <Toast
          message={toast.message}
          type={toast.type}
          onClose={() => setToast(null)}
        />
      )}

      {/* Image preview modal */}
      {previewUrl && (
        <ImagePreview url={previewUrl} onClose={() => setPreviewUrl(null)} />
      )}
    </div>
  );
}
