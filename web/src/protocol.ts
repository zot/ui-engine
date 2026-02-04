// Protocol message types and helpers
// CRC: crc-ProtocolHandler.md
// Spec: protocol.md

export type MessageType =
  | 'create'
  | 'destroy'
  | 'update'
  | 'watch'
  | 'unwatch'
  | 'error'
  | 'get'
  | 'getObjects'
  | 'poll';

export interface Message {
  type: MessageType;
  data?: unknown;
}

// Spec: protocol.md - create(id, parentId, value, properties, nowatch?, unbound?)
export interface CreateMessage {
  id: number;
  parentId?: number;
  value?: unknown;
  properties?: Record<string, string>;
  nowatch?: boolean;
  unbound?: boolean;
}

export interface DestroyMessage {
  varId: number;
}

export interface UpdateMessage {
  varId: number;
  value?: unknown;
  properties?: Record<string, string>;
}

export interface WatchMessage {
  varId: number;
}

// Spec: protocol.md - error(varId, code, description)
export interface ErrorMessage {
  varId?: number;
  code: string;        // One-word error code (e.g., "path-failure", "not-found", "unauthorized")
  description: string; // Human-readable error description
}

export interface GetMessage {
  varIds: number[];
}

export interface GetObjectsMessage {
  objIds: number[];
}

export interface PollMessage {
  wait?: string;
}

export interface VariableData {
  id: number;
  value?: unknown;
  properties?: Record<string, string>;
}

export interface ObjectData {
  obj: number;
  value: unknown;
}

export function createMessage(type: MessageType, data?: unknown): Message {
  return { type, data };
}

export function parseMessage(json: string): Message {
  return JSON.parse(json);
}

export function encodeMessage(msg: Message): string {
  return JSON.stringify(msg);
}
