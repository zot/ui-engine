// Message batching with priority support
// CRC: crc-MessageBatcher.md
// Spec: protocol.md

import { Message, UpdateMessage } from './protocol';

/** Priority levels for batching */
export enum Priority {
  High = 0,
  Medium = 1,
  Low = 2,
}

/**
 * Parse priority suffix from property name.
 * @param propertyName Property name, possibly with :high/:med/:low suffix
 * @returns Tuple of [baseName, priority]
 */
export function parsePrioritySuffix(propertyName: string): [string, Priority] {
  if (propertyName.endsWith(':high')) {
    return [propertyName.slice(0, -5), Priority.High];
  }
  if (propertyName.endsWith(':med')) {
    return [propertyName.slice(0, -4), Priority.Medium];
  }
  if (propertyName.endsWith(':low')) {
    return [propertyName.slice(0, -4), Priority.Low];
  }
  return [propertyName, Priority.Medium];
}

/** Pending change for a variable */
interface PendingChange {
  varId: number;
  value?: unknown;
  valuePriority: Priority;
  hasValue: boolean;
  properties: Map<string, string>;
  propPriorities: Map<string, Priority>;
}

/** Message batcher that groups updates by priority */
export class MessageBatcher {
  private pending: Map<number, PendingChange> = new Map();

  /** Get or create pending change for variable */
  private getOrCreate(varId: number): PendingChange {
    let pc = this.pending.get(varId);
    if (!pc) {
      pc = {
        varId,
        valuePriority: Priority.Medium,
        hasValue: false,
        properties: new Map(),
        propPriorities: new Map(),
      };
      this.pending.set(varId, pc);
    }
    return pc;
  }

  /** Queue a value change with priority */
  queueValue(varId: number, value: unknown, priority: Priority = Priority.Medium): void {
    const pc = this.getOrCreate(varId);
    pc.value = value;
    pc.valuePriority = priority;
    pc.hasValue = true;
  }

  /**
   * Queue a property change.
   * Property name can include priority suffix (e.g., "viewdefs:high")
   */
  queueProperty(varId: number, propertyName: string, value: string): void {
    const [baseName, priority] = parsePrioritySuffix(propertyName);
    const pc = this.getOrCreate(varId);
    pc.properties.set(baseName, value);
    pc.propPriorities.set(baseName, priority);
  }

  /** Queue multiple properties */
  queueProperties(varId: number, properties: Record<string, string>): void {
    for (const [name, value] of Object.entries(properties)) {
      this.queueProperty(varId, name, value);
    }
  }

  /** Check if any changes are pending */
  isEmpty(): boolean {
    return this.pending.size === 0;
  }

  /** Build and return batched messages, clearing pending state */
  flush(): Message[] {
    if (this.pending.size === 0) {
      return [];
    }

    interface BatchEntry {
      priority: Priority;
      message: Message;
    }

    const entries: BatchEntry[] = [];

    for (const pc of this.pending.values()) {
      // Group properties by priority
      const highProps: Record<string, string> = {};
      const medProps: Record<string, string> = {};
      const lowProps: Record<string, string> = {};

      for (const [name, value] of pc.properties) {
        const priority = pc.propPriorities.get(name) ?? Priority.Medium;
        switch (priority) {
          case Priority.High:
            highProps[name] = value;
            break;
          case Priority.Medium:
            medProps[name] = value;
            break;
          case Priority.Low:
            lowProps[name] = value;
            break;
        }
      }

      // Create messages for each priority level with content
      // High priority
      if (
        Object.keys(highProps).length > 0 ||
        (pc.hasValue && pc.valuePriority === Priority.High)
      ) {
        const msg = this.createUpdateMessage(
          pc.varId,
          pc.hasValue && pc.valuePriority === Priority.High,
          pc.value,
          highProps
        );
        if (msg) {
          entries.push({ priority: Priority.High, message: msg });
        }
      }

      // Medium priority
      if (
        Object.keys(medProps).length > 0 ||
        (pc.hasValue && pc.valuePriority === Priority.Medium)
      ) {
        const msg = this.createUpdateMessage(
          pc.varId,
          pc.hasValue && pc.valuePriority === Priority.Medium,
          pc.value,
          medProps
        );
        if (msg) {
          entries.push({ priority: Priority.Medium, message: msg });
        }
      }

      // Low priority
      if (
        Object.keys(lowProps).length > 0 ||
        (pc.hasValue && pc.valuePriority === Priority.Low)
      ) {
        const msg = this.createUpdateMessage(
          pc.varId,
          pc.hasValue && pc.valuePriority === Priority.Low,
          pc.value,
          lowProps
        );
        if (msg) {
          entries.push({ priority: Priority.Low, message: msg });
        }
      }
    }

    // Clear pending
    this.pending.clear();

    // Sort by priority (High=0 first, then Medium=1, then Low=2)
    entries.sort((a, b) => a.priority - b.priority);

    return entries.map((e) => e.message);
  }

  /** Create update message for a variable */
  private createUpdateMessage(
    varId: number,
    includeValue: boolean,
    value: unknown,
    properties: Record<string, string>
  ): Message | null {
    if (!includeValue && Object.keys(properties).length === 0) {
      return null;
    }

    const update: UpdateMessage = { varId };
    if (includeValue) {
      update.value = value;
    }
    if (Object.keys(properties).length > 0) {
      update.properties = properties;
    }

    return { type: 'update', data: update };
  }

  /** Flush and return as JSON string (array if multiple, object if single) */
  flushJSON(): string | null {
    const messages = this.flush();
    if (messages.length === 0) {
      return null;
    }
    if (messages.length === 1) {
      return JSON.stringify(messages[0]);
    }
    return JSON.stringify(messages);
  }
}

/**
 * Parse incoming batch (array or single message).
 * @param json JSON string that may be a single message or array
 * @returns Array of messages
 */
export function parseBatch(json: string): Message[] {
  const parsed = JSON.parse(json);
  if (Array.isArray(parsed)) {
    return parsed;
  }
  return [parsed];
}
