// Frontend outgoing message batcher with debouncing and priority sorting
// CRC: crc-FrontendOutgoingBatcher.md
// Spec: protocol.md
// Sequence: seq-frontend-outgoing-batch.md

import { Message } from './protocol';

export type Priority = 'high' | 'medium' | 'low';

interface QueuedMessage {
  message: Message;
  priority: Priority;
  order: number; // For FIFO within priority
}

const PRIORITY_ORDER: Record<Priority, number> = {
  high: 0,
  medium: 1,
  low: 2,
};

export class FrontendOutgoingBatcher {
  private pendingMessages: QueuedMessage[] = [];
  private debounceTimer: ReturnType<typeof setTimeout> | null = null;
  private insertionOrder = 0;
  private sendFn: (data: string) => void;

  readonly debounceInterval = 50; // ms

  constructor(sendFn: (data: string) => void) {
    this.sendFn = sendFn;
  }

  // Add message to pending queue with priority (debounced send)
  enqueue(msg: Message, priority: Priority = 'medium'): void {
    this.pendingMessages.push({
      message: msg,
      priority,
      order: this.insertionOrder++,
    });
    this.startDebounce();
  }

  // Add message to queue then flush immediately (for user events)
  // CRC: crc-FrontendOutgoingBatcher.md - enqueueAndFlush
  enqueueAndFlush(msg: Message, priority: Priority = 'high'): void {
    this.pendingMessages.push({
      message: msg,
      priority,
      order: this.insertionOrder++,
    });
    this.flushNow();
  }

  // Sort by priority (high -> medium -> low), FIFO within priority, send batch
  flush(): void {
    if (this.pendingMessages.length === 0) {
      return;
    }

    // Sort: first by priority, then by insertion order (FIFO)
    this.pendingMessages.sort((a, b) => {
      const priorityDiff = PRIORITY_ORDER[a.priority] - PRIORITY_ORDER[b.priority];
      if (priorityDiff !== 0) return priorityDiff;
      return a.order - b.order;
    });

    // Extract messages and send as batch
    const messages = this.pendingMessages.map((q) => q.message);
    this.pendingMessages = [];
    this.insertionOrder = 0;

    // Send as JSON array (batched format per spec)
    this.sendFn(JSON.stringify(messages));
  }

  // Start/restart 50ms debounce timer (resets on each call)
  private startDebounce(): void {
    this.cancelDebounce();
    this.debounceTimer = setTimeout(() => {
      this.debounceTimer = null;
      this.flush();
    }, this.debounceInterval);
  }

  // Cancel pending timer
  cancelDebounce(): void {
    if (this.debounceTimer !== null) {
      clearTimeout(this.debounceTimer);
      this.debounceTimer = null;
    }
  }

  // Flush immediately and cancel timer (for cleanup/disconnect)
  flushNow(): void {
    this.cancelDebounce();
    this.flush();
  }

  // Get pending message count (for testing)
  get pendingCount(): number {
    return this.pendingMessages.length;
  }
}
