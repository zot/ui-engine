# Sequence: Frontend Outgoing Batch

**Source Spec:** protocol.md
**Use Case:** Frontend throttles and batches outgoing messages with priority sorting

## Participants

- FrontendApp: Browser application
- FrontendOutgoingBatcher: Message queue with throttling
- SharedWorker: Tab coordination and WebSocket management
- WebSocketEndpoint: Server connection

## Sequence

```
     FrontendApp       FrontendOutgoingBatcher        SharedWorker        WebSocketEndpoint
        |                         |                         |                         |
        |     [create message - sent immediately]           |                         |
        |---create(props)-------->|                         |                         |
        |                         |---shouldBypassBatch?--->|                         |
        |                         |   (yes for create)      |                         |
        |                         |---sendImmediate-------->|                         |
        |                         |                         |---send(create)--------->|
        |                         |                         |                         |
        |     [other messages - batched with throttle]      |                         |
        |---update(varId,val)---->|                         |                         |
        |                         |---enqueue(msg, med)---->|                         |
        |                         |                         |                         |
        |                         |---startThrottle-------->|                         |
        |                         |   (200ms timer)         |                         |
        |                         |                         |                         |
        |---watch(varId)--------->|                         |                         |
        |                         |---enqueue(msg, med)---->|                         |
        |                         |                         |                         |
        |---update(id2,val,hi)--->|                         |                         |
        |                         |---enqueue(msg, high)--->|                         |
        |                         |                         |                         |
        |                         |     [200ms timer fires] |                         |
        |                         |---flush---------------->|                         |
        |                         |   sort: high->med->low  |                         |
        |                         |   FIFO within priority  |                         |
        |                         |                         |                         |
        |                         |---send(batch)---------->|                         |
        |                         |                         |---send([hi,med,med])--->|
        |                         |                         |                         |
```

## Notes

- `create` messages bypass batching for minimal latency
- Other messages (update, watch, unwatch, destroy) are batched
- 200ms throttle window starts on first enqueue
- Batch sorted by priority, FIFO within priority
- Empty queue does not start timer
