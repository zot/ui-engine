# FrontendOutgoingBatcher

**Source Spec:** protocol.md

## Responsibilities

### Knows
- pendingMessages: Array of queued messages with priorities
- debounceInterval: 50ms batch interval
- debounceTimer: Timer reference for pending batch send
- priorityOrder: high -> medium -> low

### Does
- enqueue: Add message to pending queue with priority, restart debounce timer
- enqueueAndFlush: Add message to queue then flush immediately (for user events)
- flush: Sort by priority (FIFO within priority), send batch, clear queue
- startDebounce: Start/restart 50ms debounce timer
- cancelDebounce: Cancel pending timer

## Collaborators

- FrontendApp: Queues protocol messages
- SharedWorker: Sends batched messages to server
- Connection: WebSocket send

## Sequences

- seq-frontend-outgoing-batch.md: Debounced batching with priority sorting

## Notes

**Batching behavior (debounce):**
- Messages queued during quiet period are batched together
- Batch sorted by priority: high -> medium -> low
- Within same priority, FIFO order preserved
- Timer restarts on each enqueue (debounce), fires after 50ms of inactivity

**Immediate flush (user events):**
- User events (action bindings) call enqueueAndFlush for immediate send
- This ensures responsive UI feedback for clicks, keypresses, etc.
- Flush sends all pending messages including the new one
