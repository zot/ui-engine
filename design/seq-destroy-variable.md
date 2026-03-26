# Sequence: Destroy Variable

**Source Spec:** protocol.md, destroy-response-batching.md
**Use Case:** Destroying a variable and its children

## Participants

- Sender: Frontend or backend initiating destroy
- Session: Frontend layer session
- Handler: Protocol message handler
- LuaBackend: Per-session backend (implements Backend interface)
- Tracker: change-tracker.Tracker instance (per-session)
- VariableStore: In-memory variable store
- Queuer: Routes notifications through OutgoingBatcher

## Sequence

```
     Sender             Handler           LuaBackend             Tracker            Queuer
        |                   |                   |                      |                |
        |---destroy(varId)->|                   |                      |                |
        |                   |                   |                      |                |
        |                   |--DestroyVariable->|                      |                |
        |                   |   (varId)         |                      |                |
        |                   |                   |---getChildren------->|                |
        |                   |                   |   (varId)            |                |
        |                   |                   |<--childIds-----------|                |
        |                   |                   |                      |                |
        |                   |                   |     [for each child, recursively]     |
        |                   |                   |---destroy(childId)-->|                |
        |                   |                   |                      |                |
        |                   |                   |---removeWatchers---->|                |
        |                   |                   |   (varId)            |                |
        |                   |                   |---delete(varId)----->|                |
        |                   |                   |                      |                |
        |                   |<--destroyed[]-----|                      |                |
        |                   |                   |                      |                |
        |                   |     [for each destroyed varId]           |                |
        |                   |---Queue(destroyNotif, [connID])----------------------------->|
        |                   |                   |                      |                |
        |                   |   [batcher accumulates; timer sends batch]                |
        |                   |                   |                      |                |
```

## Notes

- Destruction is recursive - all children destroyed first
- DestroyVariable returns list of destroyed IDs (children-first order)
- Destroy notifications are queued through OutgoingBatcher (R112)
  instead of sent directly to WebSocket
- The batcher's throttle (start timer on first Queue, accumulate on
  subsequent) batches all notifications from one incoming batch into
  a single outgoing WebSocket frame (R114)
- Watchers automatically cleaned up before deletion (per-session watchers map)
- **Per-session scope**: Variable IDs and watcher maps are session-scoped
