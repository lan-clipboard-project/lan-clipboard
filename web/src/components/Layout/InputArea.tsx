/**
 * InputArea — fixed at the bottom of the page.
 * Contains a multi-line text input, send button, and file upload button.
 * Handles paste-to-upload images and keyboard shortcuts.
 */

import { forwardRef, useState, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { createTextClip } from '../../api/clips';
import { useClipStore } from '../../store/clipStore';
import { MAX_TEXT_SIZE } from '../../utils/constants';
import { formatFileSize } from '../../utils/format';
import UploadProgress from '../Upload/UploadProgress';
import type { UploadFileProgress } from '../../types';

const textBox = 'w-full whitespace-pre-wrap break-words border px-4 py-[9px] text-sm leading-6';

interface InputAreaProps {
  onUploadFiles: (files: File[]) => void;
  uploads: UploadFileProgress[];
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export default forwardRef<HTMLDivElement, InputAreaProps>(function InputArea({ onUploadFiles, uploads, onShowToast }, ref) {
  const { t } = useTranslation();
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const addClip = useClipStore((s) => s.addClip);

  // Send text
  const handleSend = useCallback(async () => {
    const trimmed = text.trim();
    if (!trimmed || sending) return;

    if (new Blob([trimmed]).size > MAX_TEXT_SIZE) {
      onShowToast(t('input.textTooLong', { max: formatFileSize(MAX_TEXT_SIZE) }), 'error');
      return;
    }

    setSending(true);
    try {
      const clip = await createTextClip(trimmed);
      // Add immediately instead of waiting for the WebSocket echo, so the UI
      // still updates if the socket is reconnecting.
      addClip(clip);
      setText('');
    } catch {
      onShowToast(t('error.sendFailed'), 'error');
    }
    setSending(false);
  }, [text, sending, onShowToast, t, addClip]);

  // Keyboard shortcut: Ctrl/Cmd + Enter to send
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend],
  );

  // Paste handler: detect image paste
  const handlePaste = useCallback(
    (e: React.ClipboardEvent) => {
      const items = e.clipboardData.items;
      const imageFiles: File[] = [];
      for (let i = 0; i < items.length; i++) {
        if (items[i].type.startsWith('image/')) {
          const file = items[i].getAsFile();
          if (file) imageFiles.push(file);
        }
      }
      if (imageFiles.length > 0) {
        e.preventDefault();
        onUploadFiles(imageFiles);
      }
    },
    [onUploadFiles],
  );

  // File input change handler
  const handleFileChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const files = Array.from(e.target.files || []);
      if (files.length > 0) {
        onUploadFiles(files);
      }
      // Reset input so the same file can be selected again
      e.target.value = '';
    },
    [onUploadFiles],
  );

  const textByteSize = new Blob([text]).size;
  const isNearLimit = textByteSize > MAX_TEXT_SIZE * 0.9;
  const isOverLimit = textByteSize > MAX_TEXT_SIZE;

  return (
    <div ref={ref} className="relative z-20 shrink-0 border-t frosted-bar">
      {/* Upload progress indicators */}
      {uploads.length > 0 && (
        <div className="max-w-3xl mx-auto px-4 pt-2">
          <UploadProgress uploads={uploads} />
        </div>
      )}

      <div className="max-w-3xl mx-auto px-4 py-3">
        <div className="flex items-end gap-2">
          {/* File upload button */}
          <button
            onClick={() => fileInputRef.current?.click()}
            className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400"
            title={t('search.file')}
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
            </svg>
          </button>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={handleFileChange}
          />

          {/* Text input */}
          <div className="group/composer relative flex-1 min-w-0">
            {/* Mirror wrapping and trailing newlines so the field can grow with the draft. */}
            <div aria-hidden="true"
              className={`${textBox} invisible min-h-11 max-h-[140px] overflow-hidden border-transparent group-focus-within/composer:min-h-[92px] motion-safe:transition-[min-height] motion-safe:duration-300 motion-safe:ease-[cubic-bezier(0.22,1,0.36,1)]`}>
              {text}{'\u200b'}
            </div>
            <textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              onKeyDown={handleKeyDown}
              onPaste={handlePaste}
              placeholder={t('input.placeholder')}
              rows={1}
              className={`${textBox} absolute inset-0 block h-full resize-none overflow-y-auto rounded-xl border-gray-300 dark:border-gray-600 bg-gray-50/90 dark:bg-gray-800/90 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent transition-shadow placeholder-gray-500 dark:placeholder-gray-400`}
            />
          </div>

          {/* Send button */}
          <button
            onClick={handleSend}
            disabled={!text.trim() || sending || isOverLimit}
            className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-blue-500 hover:bg-blue-600 disabled:bg-gray-300 dark:disabled:bg-gray-700 text-white disabled:text-gray-500"
            title={t('input.sendHint')}
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 12L3.269 3.126A59.768 59.768 0 0121.485 12 59.77 59.77 0 013.27 20.876L5.999 12zm0 0h7.5" />
            </svg>
          </button>
        </div>
        {/* Footer hints: character count (near limit) on the left, send hint on the right */}
        <div className="flex items-center justify-between mt-1.5 text-xs">
          {isNearLimit ? (
            <span className={isOverLimit ? 'text-red-500' : 'text-amber-500'}>
              {formatFileSize(textByteSize)} / {formatFileSize(MAX_TEXT_SIZE)}
            </span>
          ) : (
            <span />
          )}
          <span className="text-gray-600 dark:text-gray-300">{t('input.sendHint')}</span>
        </div>
      </div>
    </div>
  );
});
