# FrontendOutgoingBatcher

**Source Spec:** protocol.md
**Requirements:** R40, R41

## Responsibilities

### Knows
- pendingMessages: Array of queued messages with priorities
- debounceInterval: 10ms batch interval
- debounceTimer: Timer reference for pending batch send
- priorityOrder: high -> medium -> low
- userEvent: Whether current batch contains user-triggered messages

### Does
- enqueue: Add message to pending queue with priority, restart debounce timer, set userEvent=false
- enqueueAndFlush: Add message to queue then flush immediately with userEvent=true (for user events)
- flush: Sort by priority (FIFO within priority), send batch wrapper with userEvent flag, clear queue
- startDebounce: Start/restart 10ms debounce timer
- cancelDebounce: Cancel pending timer
- ensureDebounceStarted: Start timer if not running (called before processing incoming)

## Collaborators

- FrontendApp: Queues protocol messages
- SharedWorker: Sends batched messages to server
- Connection: WebSocket send, handles incoming batches

## Sequences

- seq-frontend-outgoing-batch.md: Debounced batching with priority sorting and userEvent flag

## Notes

**Batch format:**
```json
{"userEvent": true, "messages": [{...}, {...}]}
```

**Batching behavior (debounce):**
- Messages queued during quiet period are batched together
- Batch sorted by priority: high -> medium -> low
- Within same priority, FIFO order preserved
- Timer restarts on each enqueue (debounce), fires after 10ms of inactivity
- Non-user-triggered batches sent with `userEvent: false`

**Immediate flush (user events):**
- User events (action bindings) call enqueueAndFlush for immediate send
- This ensures responsive UI feedback for clicks, keypresses, etc.
- Flush sends all pending messages with `userEvent: true`
- Server receives flag and flushes responses immediately
