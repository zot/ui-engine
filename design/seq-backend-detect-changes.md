# Sequence: Backend Detect Changes

**Source Spec:** libraries.md (Change detection), protocol.md
**Use Case:** Per-session change detection after processing message batch

## Participants

- WebSocketEndpoint: Receives message batch from frontend
- Session: Frontend layer session
- LuaBackend: Per-session Lua backend
- Tracker: change-tracker.Tracker instance
- MessageRelay: Sends updates to watchers

## Sequence

```
   WebSocketEndpoint        Session            LuaBackend             Tracker          MessageRelay
        |                      |                   |                   |                   |
        |---handleBatch------->|                   |                   |                   |
        |   (messages[])       |                   |                   |                   |
        |                      |                   |                   |                   |
        |                      |--HandleMessage--->|                   |                   |
        |                      |   (foreach msg)   |                   |                   |
        |                      |                   |                   |                   |
        |                      |                   |--[process msgs]-->|                   |
        |                      |                   |   (create/update/ | (self)           |
        |                      |                   |    watch/etc)     |                   |
        |                      |                   |                   |                   |
        |                      |                   |   [after batch]   |                   |
        |                      |                   |--DetectChanges()->|                   |
        |                      |                   |                   |                   |
        |                      |                   |                   |--computeValues-->|
        |                      |                   |                   |   (all watched)  | (self)
        |                      |                   |                   |                   |
        |                      |                   |                   |--compareCache--->|
        |                      |                   |                   |   (find changes) | (self)
        |                      |                   |                   |                   |
        |                      |                   |                   |   [for each changed var]
        |                      |                   |                   |--getWatchers---->|
        |                      |                   |                   |   (watchers map) | (LuaBackend)
        |                      |                   |                   |                   |
        |                      |                   |                   |--send(update)--->|
        |                      |                   |                   |                   |
        |                      |                   |                   |                   |
        |                      |                   |<--ok--------------|                   |
        |                      |                   |                   |                   |
        |                      |<--ok--------------|                   |                   |
        |                      |                   |                   |                   |
        |<--ok-----------------|                   |                   |                   |
        |                      |                   |                   |                   |
```

## Notes

- Change detection is per-session (each LuaBackend has its own Tracker)
- Tracker computes current values for all watched variables via path resolution
- Values are compared to cached values to detect changes
- Updates are sent only to watchers within this session (watchers map is per-session)
- No cross-session interference because variable IDs and maps are session-scoped
- Watcher management is handled by LuaBackend per-session, avoiding the collisions that occurred with global maps
