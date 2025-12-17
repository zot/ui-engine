# Sequence: Destroy Variable

**Source Spec:** protocol.md
**Use Case:** Destroying a variable and its children

## Participants

- Sender: Frontend or backend initiating destroy
- Session: Frontend layer session
- LuaBackend: Per-session backend (implements Backend interface)
- Tracker: change-tracker.Tracker instance (per-session)
- VariableStore: Variable storage

## Sequence

```
     Sender              Session            LuaBackend             Tracker           VariableStore
        |                   |                   |                      |                      |
        |---destroy(varId)->|                   |                      |                      |
        |                   |                   |                      |                      |
        |                   |--HandleMessage--->|                      |                      |
        |                   |                   |                      |                      |
        |                   |                   |---getChildren------->|                      |
        |                   |                   |   (varId)            |                      |
        |                   |                   |                      |                      |
        |                   |                   |<--childIds-----------|                      |
        |                   |                   |                      |                      |
        |                   |                   |     [for each child, recursively]           |
        |                   |                   |---destroy(childId)-->|                      |
        |                   |                   |                      | (self)               |
        |                   |                   |                      |                      |
        |                   |                   |---removeWatchers---->|                      |
        |                   |                   |   (varId)            | (self)               |
        |                   |                   |                      |                      |
        |                   |                   |     [if had watchers]|                      |
        |                   |                   |---Unwatch(varId)---->|                      |
        |                   |                   |                      |                      |
        |                   |                   |<--ok-----------------|                      |
        |                   |                   |                      |                      |
        |                   |                   |---delete(varId)----->|                      |
        |                   |                   |                      |---delete()---------->|
        |                   |                   |                      |                      |
```

## Notes

- Destruction is recursive - all children destroyed first
- Watchers automatically cleaned up before deletion (per-session watchers map)
- Unwatch removes from change-tracker if variable had observers
- **Per-session scope**: Variable IDs and watcher maps are session-scoped
- Source of truth holder removes from storage
