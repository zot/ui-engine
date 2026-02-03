# ServerOutgoingBatcher

**Source Spec:** protocol.md

## Responsibilities

### Knows
- pendingMessages: Map of sessionID -> queued messages
- debounceTimers: Map of sessionID -> timer reference
- debounceInterval: 10ms batch interval
- sendFn: Function to send messages to frontend

### Does
- Queue: Add messages to session's pending queue (starts timer if not running)
- FlushNow: Send all pending messages for session immediately
- EnsureDebounceStarted: Start debounce timer if not running (called before processing)
- startDebounce: Start/restart 10ms debounce timer for session
- cancelDebounce: Cancel pending timer for session

## Collaborators

- Server: Calls Queue/FlushNow based on userEvent flag
- WebSocketEndpoint: Sends batched messages to frontend

## Sequences

- seq-frontend-outgoing-batch.md: Server-side response batching

## Notes

**Per-session queuing:**
- Each session has its own pending message queue
- Each session has its own debounce timer
- Timers are independent - flushing one session doesn't affect others

**Immediate flush (user events):**
- When server receives `userEvent=true` batch, AfterBatch calls FlushNow
- Responses to user actions are sent immediately for responsive UI

**Debounced flush (non-user events):**
- When server receives `userEvent=false` batch, AfterBatch calls Queue
- 10ms timer coalesces rapid backend changes into single batch
- Reduces network traffic for programmatic updates

**Pre-start optimization:**
- EnsureDebounceStarted starts timer before processing, so debounce runs concurrently
- If processing takes time, timer is already running, reducing total latency
- Queue preserves existing timer, doesn't restart if pre-started
