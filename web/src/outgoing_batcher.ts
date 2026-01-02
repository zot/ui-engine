// Frontend outgoing message batcher with throttling and priority sorting
// CRC: crc-FrontendOutgoingBatcher.md
// Spec: protocol.md
// Sequence: seq-frontend-outgoing-batch.md

import { Message, encodeMessage } from './protocol';

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
  private throttleTimer: ReturnType<typeof setTimeout> | null = null;
  private insertionOrder = 0;
  private sendFn: (data: string) => void;

  readonly throttleInterval = 200; // ms

  constructor(sendFn: (data: string) => void) {
    this.sendFn = sendFn;
  }

  // Check if message type bypasses batching
  shouldBypassBatch(msg: Message): boolean {
    return msg.type === 'create';
  }

  // Add message to pending queue with priority
  enqueue(msg: Message, priority: Priority = 'medium'): void {
    this.pendingMessages.push({
      message: msg,
      priority,
      order: this.insertionOrder++,
    });
    this.startThrottle();
  }

  // Send message immediately, bypassing batch queue
  sendImmediate(msg: Message): void {
    this.sendFn(encodeMessage(msg));
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

  // Start 200ms timer if not running
  private startThrottle(): void {
    if (this.throttleTimer !== null) {
      return; // Timer already running
    }
    this.throttleTimer = setTimeout(() => {
      this.throttleTimer = null;
      this.flush();
    }, this.throttleInterval);
  }

  // Cancel pending timer
  cancelThrottle(): void {
    if (this.throttleTimer !== null) {
      clearTimeout(this.throttleTimer);
      this.throttleTimer = null;
    }
  }

  // Flush immediately and cancel timer (for cleanup/disconnect)
  flushNow(): void {
    this.cancelThrottle();
    this.flush();
  }

  // Get pending message count (for testing)
  get pendingCount(): number {
    return this.pendingMessages.length;
  }
}
