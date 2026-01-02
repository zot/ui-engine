# FrontendOutgoingBatcher

**Source Spec:** protocol.md

## Responsibilities

### Knows
- pendingMessages: Array of queued messages with priorities
- throttleInterval: 200ms batch interval
- throttleTimer: Timer reference for pending batch send
- priorityOrder: high -> medium -> low

### Does
- enqueue: Add message to pending queue with priority (default: medium)
- sendImmediate: Send message bypassing batch queue (for create messages)
- shouldBypassBatch: Check if message type bypasses batching (create messages)
- flush: Sort by priority (FIFO within priority), send batch, clear queue
- startThrottle: Start 200ms timer if not running
- cancelThrottle: Cancel pending timer

## Collaborators

- FrontendApp: Queues protocol messages
- SharedWorker: Sends batched messages to server
- Connection: WebSocket send

## Sequences

- seq-frontend-outgoing-batch.md: Throttled batching with priority sorting

## Notes

**Batching behavior:**
- Messages queued during 200ms window are batched together
- Batch sorted by priority: high -> medium -> low
- Within same priority, FIFO order preserved
- Timer starts on first enqueue, fires at 200ms

**Immediate send (bypass batching):**
- `create` messages sent immediately to minimize latency
- Immediate messages do not reset the batch timer
