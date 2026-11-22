/**
 * WebSocket client with auto-reconnect and exponential backoff.
 * Handles real-time clip events from the server.
 */

import type { ConnectionStatus, WSMessage } from '../types';
import { WS_INITIAL_DELAY_MS, WS_MAX_DELAY_MS, WS_BACKOFF_FACTOR } from '../utils/constants';

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private status: ConnectionStatus = 'disconnected';
  private reconnectDelay = WS_INITIAL_DELAY_MS;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private intentionalClose = false;

  /** Callback invoked when a clip event is received */
  public onMessage: ((msg: WSMessage) => void) | null = null;

  /** Callback invoked when connection status changes */
  public onStatusChange: ((status: ConnectionStatus) => void) | null = null;

  /** Build the WebSocket URL based on the current page origin */
  private getWsUrl(): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${window.location.host}/ws`;
  }

  /** Connect to the WebSocket server */
  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN || this.ws?.readyState === WebSocket.CONNECTING) {
      return;
    }

    this.intentionalClose = false;
    this.setStatus('connecting');

    try {
      this.ws = new WebSocket(this.getWsUrl());
    } catch {
      this.setStatus('disconnected');
      this.scheduleReconnect();
      return;
    }

    this.ws.onopen = () => {
      this.setStatus('connected');
      this.reconnectDelay = WS_INITIAL_DELAY_MS; // Reset backoff on successful connect
      this.startPing();
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        // Ignore pong messages
        if (data.type === 'pong') return;
        this.onMessage?.(data as WSMessage);
      } catch {
        // Ignore malformed messages
      }
    };

    this.ws.onclose = () => {
      this.stopPing();
      this.setStatus('disconnected');
      if (!this.intentionalClose) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = () => {
      // Error event is always followed by close event, so reconnect is handled there
    };
  }

  /** Disconnect from the WebSocket server */
  disconnect(): void {
    this.intentionalClose = true;
    this.clearReconnectTimer();
    this.stopPing();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.setStatus('disconnected');
  }

  /** Get current connection status */
  getStatus(): ConnectionStatus {
    return this.status;
  }

  /** Update status and notify listener */
  private setStatus(status: ConnectionStatus): void {
    if (this.status !== status) {
      this.status = status;
      this.onStatusChange?.(status);
    }
  }

  /** Schedule a reconnection attempt with exponential backoff */
  private scheduleReconnect(): void {
    this.clearReconnectTimer();
    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, this.reconnectDelay);

    // Increase delay for next attempt, capped at max
    this.reconnectDelay = Math.min(this.reconnectDelay * WS_BACKOFF_FACTOR, WS_MAX_DELAY_MS);
  }

  /** Clear pending reconnection timer */
  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  /** Start periodic ping to keep connection alive */
  private startPing(): void {
    this.stopPing();
    this.pingTimer = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'ping' }));
      }
    }, 30000); // Ping every 30 seconds
  }

  /** Stop ping timer */
  private stopPing(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
  }
}
