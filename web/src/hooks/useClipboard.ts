/**
 * Hook for copying text to the system clipboard.
 * Uses the Clipboard API with a fallback.
 */

import { useCallback, useState } from 'react';

export function useClipboard() {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = useCallback(async (text: string): Promise<boolean> => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
      return true;
    } catch {
      // Fallback for older browsers or insecure contexts
      try {
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
        setCopied(true);
        setTimeout(() => setCopied(false), 2000);
        return true;
      } catch {
        return false;
      }
    }
  }, []);

  return { copied, copyToClipboard };
}
