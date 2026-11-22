/**
 * Hook for managing the WebSocket connection.
 * Dispatches incoming messages to the Zustand store.
 */

import { useEffect, useRef, useState } from 'react';
import { WebSocketClient } from '../api/ws';
import { useClipStore } from '../store/clipStore';
import type { Clip, ConnectionStatus, WSMessage } from '../types';

export function useWebSocket(): { status: ConnectionStatus } {
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const wsRef = useRef<WebSocketClient | null>(null);

  const addClip = useClipStore((s) => s.addClip);
  const removeClip = useClipStore((s) => s.removeClip);
  const updateClip = useClipStore((s) => s.updateClip);
  const clearClips = useClipStore((s) => s.clearClips);

  useEffect(() => {
    const client = new WebSocketClient();
    wsRef.current = client;

    client.onStatusChange = (newStatus) => {
      setStatus(newStatus);
    };

    client.onMessage = (msg: WSMessage) => {
      switch (msg.type) {
        case 'clip_created':
          addClip(msg.data as Clip);
          break;
        case 'clip_deleted':
          removeClip((msg.data as { id: string }).id);
          break;
        case 'clip_updated':
          updateClip(msg.data as Clip);
          break;
        case 'clips_cleared':
          clearClips();
          break;
      }
    };

    client.connect();

    return () => {
      client.disconnect();
    };
  }, [addClip, removeClip, updateClip, clearClips]);

  return { status };
}
