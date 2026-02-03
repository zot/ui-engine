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

// Batch wrapper format sent to server
interface BatchWrapper {
  userEvent: boolean;
  messages: Message[];
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
  private userEvent = false; // True if batch contains user-triggered messages
  private sendFn: (data: string) => void;

  readonly debounceInterval = 10; // ms

  constructor(sendFn: (data: string) => void) {
    this.sendFn = sendFn;
  }

  // Add message to pending queue with priority (debounced send)
  // userEvent=false for server-triggered changes
  enqueue(msg: Message, priority: Priority = 'medium'): void {
    this.pendingMessages.push({
      message: msg,
      priority,
      order: this.insertionOrder++,
    });
    // Don't set userEvent - this is a non-user event
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
    this.userEvent = true; // Mark batch as user-triggered
    this.flushNow();
  }

  // Sort by priority (high -> medium -> low), FIFO within priority, send batch
  private flush(): void {
    if (this.pendingMessages.length === 0) {
      this.userEvent = false;
      return;
    }

    // Sort: first by priority, then by insertion order (FIFO)
    this.pendingMessages.sort((a, b) => {
      const priorityDiff = PRIORITY_ORDER[a.priority] - PRIORITY_ORDER[b.priority];
      if (priorityDiff !== 0) return priorityDiff;
      return a.order - b.order;
    });

    // Extract messages
    const messages = this.pendingMessages.map((q) => q.message);

    // Build batch wrapper with userEvent flag
    const batch: BatchWrapper = {
      userEvent: this.userEvent,
      messages,
    };

    // Clear state
    this.pendingMessages = [];
    this.insertionOrder = 0;
    this.userEvent = false;

    // Send as JSON wrapper (new batched format per spec)
    this.sendFn(JSON.stringify(batch));
  }

  // Start/restart 10ms debounce timer (resets on each call)
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
