// WebSocket connection management
// CRC: crc-WebSocketEndpoint.md, crc-SharedWorker.md
// Spec: interfaces.md

import { Message, Response, CreateResponse, encodeMessage, UpdateMessage, ErrorMessage } from './protocol';
import { Variable } from './variable';
import { FrontendOutgoingBatcher, Priority } from './outgoing_batcher';
import type { Widget } from './binding';

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
  private requestId = 1;
  // Callback for response handling (create responses)
  // Map of requestId -> callback for correlating out-of-order responses
  private createCallbacks: Map<number, (resp: Response<CreateResponse>) => void> = new Map();
  // Outgoing message batcher (50ms debounce, priority sorting)
  // Spec: protocol.md - Frontend outgoing batching
  private batcher: FrontendOutgoingBatcher;

  constructor(sessionId: string) {
    this.sessionId = sessionId;
    this.batcher = new FrontendOutgoingBatcher((data) => this.sendRaw(data));
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
          const data = JSON.parse(event.data);
          //console.log('RECEIVED ', data)
          // Check if it's a Response (has result, error, or pending) vs Message (has type)
          if ('result' in data || ('error' in data && !('type' in data)) || 'pending' in data || 'id' in data) {
            console.log('RECEIVED RESPONSE', JSON.stringify(data))
            this.handleResponse(data as Response<CreateResponse>);
          } else {
            console.log('RECEIVED MESSAGE', JSON.stringify(data))
            this.handleMessage(data as Message);
          }
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

  private handleResponse(resp: Response<CreateResponse>): void {
    // Handle pending messages in the response
    //if (resp.pending) {
    //  resp.pending.forEach((msg) => this.handleMessage(msg));
    //}
    // Only call the callback for create responses (those with id in result)
    // Use requestId to correlate responses (supports out-of-order delivery)
    const result = resp.result;
    if (result && typeof result === 'object' && 'id' in result && result.requestId) {
      const callback = this.createCallbacks.get(result.requestId);
      if (callback) {
        this.createCallbacks.delete(result.requestId);
        callback(resp);
      }
    }
  }

  // Send raw data directly to WebSocket (used by batcher)
  private sendRaw(data: string): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(data);
    } else {
      console.error('WebSocket not connected');
    }
  }

  // Send a message, routing through batcher for debouncing
  // Set immediate=true for user events that need instant feedback
  // Spec: protocol.md - Frontend outgoing batching
  // CRC: crc-FrontendOutgoingBatcher.md - debounce vs immediate flush
  send(msg: Message, priority: Priority = 'medium', immediate = false): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.error('WebSocket not connected');
      return;
    }

    if (immediate) {
      this.batcher.enqueueAndFlush(msg, priority);
    } else {
      this.batcher.enqueue(msg, priority);
    }
  }

  // Send a message and wait for a response (for create operations)
  // Uses requestId to correlate responses (supports out-of-order delivery)
  sendAndAwaitResponse(msg: Message): Promise<Response<CreateResponse>> {
    return new Promise((resolve, reject) => {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
        reject(new Error('WebSocket not connected'));
        return;
      }
      // Generate unique requestId and add to message data
      const requestId = this.requestId++;
      const data = (msg.data || {}) as Record<string, unknown>;
      data.requestId = requestId;
      msg.data = data;
      // Register callback by requestId
      this.createCallbacks.set(requestId, resolve);
      this.ws.send(encodeMessage(msg));
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
    // Flush pending messages before closing
    this.batcher.flushNow();
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

//export type Variable = { value: unknown; properties: Record<string, string> };

// Callback types for watchers
export type ValueCallback = (v: Variable, value?: unknown, props?: Record<string, string>) => void;
export type ErrorCallback = (error: VariableError | null) => void;

// Variable store for client-side caching
// CRC: crc-VariableStore.md
export class VariableStore {
  private variables: Map<number, Variable> = new Map();
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

  private handleUpdate(varId: number, value?: unknown, properties?: Record<string, string>, parentId?: number): void {
    let existing = this.variables.get(varId);
    if (!existing) {
      existing = { varId, value: undefined, properties: {} } as Variable;
      if (parentId !== undefined) {
        existing.parentId = parentId
      }
      this.variables.set(varId, existing);
    }

    console.log('handleUpdate varId', varId, 'parentId', existing.parentId)
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
      watchers.forEach((w) => w(existing, value, properties));
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

  watch(varId: number, callback: ValueCallback, send?: boolean): () => void {
    // Add to local watchers
    let watchers = this.watchers.get(varId);
    if (!watchers) {
      watchers = new Set();
      this.watchers.set(varId, watchers);

      if (send) {
        // Send watch message to server
        this.connection.send({ type: 'watch', data: { varId } });
      }
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

  get(varId: number): Variable | undefined {
    return this.variables.get(varId);
  }

  // Create a variable and return the assigned ID
  // CRC: crc-Variable.md - widget reference set at creation time
  async create(options: {
    parentId?: number;
    value?: unknown;
    properties?: Record<string, string>;
    nowatch?: boolean;
    unbound?: boolean;
    widget?: Widget;  // Widget that created this variable
  }): Promise<number> {
    console.log('SENDING CREATE')
    const resp = await this.connection.sendAndAwaitResponse({ type: 'create', data: options });
    if (resp.error) {
      throw new Error(resp.error);
    }
    if (!resp.result) {
      throw new Error('No result from create');
    }
    if (resp.pending?.length) {
      const data = resp.pending[0].data as any as UpdateMessage;

      console.log('CREATE RESPONSE', resp.result.id, resp)
      const variable = { varId: data.varId, value: undefined, properties: options.properties || {} } as Variable;
      if (options.parentId !== undefined) {
        variable.parentId = options.parentId
      }
      if (options.widget) {
        variable.widget = options.widget
      }
      this.variables.set(data.varId, variable);
      setTimeout(()=> {
        console.log('UPDATE FROM CREATE RESPONSE', data.varId, resp)
        this.handleUpdate(data.varId, data.value, data.properties, options.parentId)
      })
    }
    return resp.result.id;
  }

  // Spec: viewdefs.md - Frontend Update Behavior
  // MUST set local value before sending to backend
  // Spec: viewdefs.md - Duplicate update suppression
  // CRC: crc-FrontendOutgoingBatcher.md - actions flush immediately
  update(varId: number, value?: unknown, properties?: Record<string, string>): void {
    const variable = this.variables.get(varId);

    // Duplicate update suppression: bindings without access=action or access=w
    // should not send an update if the value hasn't changed
    const access = variable?.properties?.access;
    const isAction = access === 'action';
    if (variable && value !== undefined) {
      const shouldAlwaysSend = isAction || access === 'w';
      if (!shouldAlwaysSend && variable.value === value) {
        // Value unchanged and not an action/write-only binding - skip update
        return;
      }
    }

    // Set local cache first (without triggering watchers - this is outgoing, not incoming)
    if (variable) {
      if (value !== undefined) {
        variable.value = value;
      }
      if (properties) {
        variable.properties = { ...variable.properties, ...properties };
      }
    }
    // Send to backend - actions flush immediately for responsive UI
    this.connection.send({ type: 'update', data: { varId, value, properties } }, 'medium', isAction);
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
