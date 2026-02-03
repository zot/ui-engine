# ServerOutgoingBatcher

**Source Spec:** protocol.md
**Requirements:** R39, R42

## Responsibilities

### Knows
- pendingUpdates: Map of sessionID -> queued updates (message + watchers)
- debounceTimers: Map of sessionID -> timer reference
- debounceInterval: 10ms batch interval
- sender: MessageSender collaborator (WebSocketEndpoint)

### Does
- Queue: Add message to session's pending queue with watchers (starts timer if not running)
- FlushNow: Send all pending messages for session immediately
- EnsureDebounceStarted: Start debounce timer if not running (called before processing)
- flushSession: Group messages by connection, send one JSON array batch per connection

## Collaborators

- Server: Calls Queue/FlushNow based on userEvent flag
- WebSocketEndpoint: Sends batched messages to frontend via MessageSender interface

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
