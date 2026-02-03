# Sequence: Frontend Outgoing Batch

**Source Spec:** protocol.md
**Use Case:** Frontend and server batch messages with userEvent flag for responsive UI

## Participants

- FrontendApp: Browser application
- FrontendOutgoingBatcher: Message queue with throttling and userEvent tracking
- SharedWorker: Tab coordination and WebSocket management
- WebSocketEndpoint: Server connection
- Server: Message processing and change detection
- ServerOutgoingBatcher: Server-side response batching

## Sequence: User Event (Immediate)

```
     FrontendApp       FrontendOutgoingBatcher        SharedWorker        WebSocketEndpoint
        |                         |                         |                         |
        |     [user clicks button - immediate flush]        |                         |
        |---action(update)------->|                         |                         |
        |                         |---enqueueAndFlush------>|                         |
        |                         |   userEvent=true        |                         |
        |                         |---send----------------->|                         |
        |                         |   {"userEvent":true,    |                         |
        |                         |    "messages":[...]}    |---send(batch)---------->|
        |                         |                         |                         |
```

```
WebSocketEndpoint        Server              ServerOutgoingBatcher
        |                   |                         |
        |---processMsg----->|                         |
        |   userEvent=true  |                         |
        |                   |---AfterBatch----------->|
        |                   |   (userEvent=true)      |
        |                   |                         |---Queue(updates)
        |                   |                         |---FlushNow()
        |<--send(updates)---|-------------------------|
        |                   |                         |
```

## Sequence: Non-User Event (Debounced)

```
     FrontendApp       FrontendOutgoingBatcher        SharedWorker        WebSocketEndpoint
        |                         |                         |                         |
        |     [server-triggered update - debounced]         |                         |
        |---update(varId,val)---->|                         |                         |
        |                         |---enqueue(msg, med)---->|                         |
        |                         |---startDebounce-------->|                         |
        |                         |   (10ms timer)          |                         |
        |                         |                         |                         |
        |---watch(varId)--------->|                         |                         |
        |                         |---enqueue(msg, med)---->|                         |
        |                         |   (timer restarts)      |                         |
        |                         |                         |                         |
        |                         |     [10ms timer fires]  |                         |
        |                         |---flush---------------->|                         |
        |                         |   userEvent=false       |                         |
        |                         |   sort: high->med->low  |                         |
        |                         |                         |                         |
        |                         |---send----------------->|                         |
        |                         |   {"userEvent":false,   |---send(batch)---------->|
        |                         |    "messages":[...]}    |                         |
```

```
WebSocketEndpoint        Server              ServerOutgoingBatcher
        |                   |                         |
        |---EnsureDebounce------------------------>|
        |   (pre-start timer before processing)    |---startDebounce(10ms)
        |                   |                         |
        |---processMsg----->|                         |   [timer running
        |   userEvent=false |                         |    concurrently]
        |                   |---AfterBatch----------->|
        |                   |   (userEvent=false)     |
        |                   |                         |---Queue(updates)
        |                   |                         |   (timer preserved)
        |                   |                         |
        |                   |     [10ms timer fires]  |
        |<--send(updates)---|-------------------------|---flush()
        |                   |                         |
```

## Notes

- User events (clicks, keypresses) flush immediately on both ends
- Non-user events (server updates) are debounced with 10ms interval
- Batch format: `{"userEvent": bool, "messages": [...]}`
- Server extracts userEvent flag and passes to AfterBatch
- ServerOutgoingBatcher maintains per-session queues and timers
- Pre-start optimization: For non-user events, debounce timer starts before processing so it runs concurrently with message handling, reducing total latency
