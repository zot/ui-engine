// WebSocket connection management
// CRC: crc-WebSocketEndpoint.md, crc-SharedWorker.md
// Spec: interfaces.md

import { Message, Response, encodeMessage, parseMessage, UpdateMessage, ErrorMessage } from './protocol';

export type MessageHandler = (msg: Message) => void;
export type ErrorHandler = (error: string) => void;
export type ConnectionHandler = () => void;

export class Connection {
  private ws: WebSocket | null = null;
  private sessionId: string;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private messageHandlers: MessageHandler[] = [];
  private errorHandlers: ErrorHandler[] = [];
  private connectHandlers: ConnectionHandler[] = [];
  private disconnectHandlers: ConnectionHandler[] = [];
  private pendingRequests: Map<number, (response: Response) => void> = new Map();
  private requestId = 0;

  constructor(sessionId: string) {
    this.sessionId = sessionId;
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const url = `${protocol}//${window.location.host}/ws/${this.sessionId}`;

      this.ws = new WebSocket(url);

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        this.connectHandlers.forEach((h) => h());
        resolve();
      };

      this.ws.onmessage = (event) => {
        try {
          const msg = parseMessage(event.data);
          this.handleMessage(msg);
        } catch (e) {
          console.error('Failed to parse message:', e);
        }
      };

      this.ws.onerror = (event) => {
        console.error('WebSocket error:', event);
        reject(new Error('WebSocket connection failed'));
      };

      this.ws.onclose = () => {
        this.disconnectHandlers.forEach((h) => h());
        this.attemptReconnect();
      };
    });
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.errorHandlers.forEach((h) => h('Max reconnection attempts reached'));
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    setTimeout(() => {
      this.connect().catch(() => {
        // Will retry in onclose handler
      });
    }, delay);
  }

  private handleMessage(msg: Message): void {
    // Notify all handlers
    this.messageHandlers.forEach((h) => h(msg));
  }

  send(msg: Message): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(encodeMessage(msg));
    } else {
      console.error('WebSocket not connected');
    }
  }

  async sendRequest<T>(msg: Message): Promise<Response<T>> {
    return new Promise((resolve) => {
      const id = this.requestId++;
      this.pendingRequests.set(id, resolve as (response: Response) => void);
      this.send(msg);
    });
  }

  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.push(handler);
    return () => {
      const idx = this.messageHandlers.indexOf(handler);
      if (idx >= 0) this.messageHandlers.splice(idx, 1);
    };
  }

  onError(handler: ErrorHandler): () => void {
    this.errorHandlers.push(handler);
    return () => {
      const idx = this.errorHandlers.indexOf(handler);
      if (idx >= 0) this.errorHandlers.splice(idx, 1);
    };
  }

  onConnect(handler: ConnectionHandler): () => void {
    this.connectHandlers.push(handler);
    return () => {
      const idx = this.connectHandlers.indexOf(handler);
      if (idx >= 0) this.connectHandlers.splice(idx, 1);
    };
  }

  onDisconnect(handler: ConnectionHandler): () => void {
    this.disconnectHandlers.push(handler);
    return () => {
      const idx = this.disconnectHandlers.indexOf(handler);
      if (idx >= 0) this.disconnectHandlers.splice(idx, 1);
    };
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}

// Error state for a variable
// Spec: protocol.md - error conditions persist until cleared
export interface VariableError {
  code: string;
  description: string;
}

// Callback types for watchers
export type ValueCallback = (value: unknown, props: Record<string, string>) => void;
export type ErrorCallback = (error: VariableError | null) => void;

// Variable store for client-side caching
// CRC: crc-VariableStore.md
export class VariableStore {
  private variables: Map<number, { value: unknown; properties: Record<string, string> }> = new Map();
  private errors: Map<number, VariableError> = new Map(); // Error state per variable
  private watchers: Map<number, Set<ValueCallback>> = new Map();
  private errorWatchers: Map<number, Set<ErrorCallback>> = new Map();
  private connection: Connection;

  constructor(connection: Connection) {
    this.connection = connection;

    // Listen for updates and errors
    connection.onMessage((msg) => {
      if (msg.type === 'update') {
        const data = msg.data as UpdateMessage;
        this.handleUpdate(data.varId, data.value, data.properties);
      } else if (msg.type === 'destroy') {
        const data = msg.data as { varId: number };
        this.handleDestroy(data.varId);
      } else if (msg.type === 'error') {
        const data = msg.data as ErrorMessage;
        if (data.varId !== undefined) {
          this.handleError(data.varId, data.code, data.description);
        }
      }
    });
  }

  private handleUpdate(varId: number, value?: unknown, properties?: Record<string, string>): void {
    let existing = this.variables.get(varId);
    if (!existing) {
      existing = { value: undefined, properties: {} };
      this.variables.set(varId, existing);
    }

    if (value !== undefined) {
      existing.value = value;
    }
    if (properties) {
      existing.properties = { ...existing.properties, ...properties };
    }

    // Clear error on successful update (spec: error clears on successful operation)
    if (this.errors.has(varId)) {
      this.errors.delete(varId);
      this.notifyErrorWatchers(varId, null);
    }

    // Notify watchers
    const watchers = this.watchers.get(varId);
    if (watchers) {
      watchers.forEach((w) => w(existing!.value, existing!.properties));
    }
  }

  private handleError(varId: number, code: string, description: string): void {
    const error: VariableError = { code, description };
    this.errors.set(varId, error);
    this.notifyErrorWatchers(varId, error);
  }

  private notifyErrorWatchers(varId: number, error: VariableError | null): void {
    const watchers = this.errorWatchers.get(varId);
    if (watchers) {
      watchers.forEach((w) => w(error));
    }
  }

  private handleDestroy(varId: number): void {
    this.variables.delete(varId);
    this.watchers.delete(varId);
  }

  watch(varId: number, callback: ValueCallback): () => void {
    // Add to local watchers
    let watchers = this.watchers.get(varId);
    if (!watchers) {
      watchers = new Set();
      this.watchers.set(varId, watchers);

      // Send watch message to server
      this.connection.send({ type: 'watch', data: { varId } });
    }
    watchers.add(callback);

    // Return unwatch function
    return () => {
      watchers!.delete(callback);
      if (watchers!.size === 0) {
        this.watchers.delete(varId);
        this.connection.send({ type: 'unwatch', data: { varId } });
      }
    };
  }

  // Watch for error state changes on a variable
  watchErrors(varId: number, callback: ErrorCallback): () => void {
    let watchers = this.errorWatchers.get(varId);
    if (!watchers) {
      watchers = new Set();
      this.errorWatchers.set(varId, watchers);
    }
    watchers.add(callback);

    // Immediately notify of current error state
    const currentError = this.errors.get(varId) ?? null;
    callback(currentError);

    return () => {
      watchers!.delete(callback);
      if (watchers!.size === 0) {
        this.errorWatchers.delete(varId);
      }
    };
  }

  // Get current error state for a variable
  getError(varId: number): VariableError | null {
    return this.errors.get(varId) ?? null;
  }

  get(varId: number): { value: unknown; properties: Record<string, string> } | undefined {
    return this.variables.get(varId);
  }

  create(options: {
    parentId?: number;
    value?: unknown;
    properties?: Record<string, string>;
    nowatch?: boolean;
    unbound?: boolean;
  }): void {
    this.connection.send({ type: 'create', data: options });
  }

  update(varId: number, value?: unknown, properties?: Record<string, string>): void {
    this.connection.send({ type: 'update', data: { varId, value, properties } });
  }

  destroy(varId: number): void {
    this.connection.send({ type: 'destroy', data: { varId } });
  }

  // Send an error message for a variable (used for path-failure, etc.)
  // Spec: protocol.md - error(varId, code, description)
  sendError(varId: number, code: string, description: string): void {
    this.connection.send({ type: 'error', data: { varId, code, description } });
    // Also update local error state
    this.handleError(varId, code, description);
  }
}
