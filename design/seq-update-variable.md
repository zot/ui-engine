# Sequence: Update Variable

**Source Spec:** protocol.md
**Use Case:** Updating variable value and/or properties

## Participants

- Sender: Frontend or backend initiating update
- Session: Frontend layer session
- LuaBackend: Per-session backend (implements Backend interface)
- VariableStore: In-memory variable store
- MessageRelay: Message forwarding

## Sequence

```
     Sender              Session            LuaBackend           VariableStore         MessageRelay
        |                   |                   |                      |                      |
        |--update(varId,--->|                   |                      |                      |
        |   value?,props?)  |                   |                      |                      |
        |                   |                   |                      |                      |
        |                   |--HandleMessage--->|                      |                      |
        |                   |                   |                      |                      |
        |                   |                   |---parseProps-------->|                      |
        |                   |                   |   (extract priority) | (self)               |
        |                   |                   |                      |                      |
        |                   |                   |---get(varId)-------->|                      |
        |                   |                   |                      |                      |
        |                   |                   |<--variable-----------|                      |
        |                   |                   |                      |                      |
        |                   |                   |     [if value provided]                     |
        |                   |                   |---setValue---------->|                      |
        |                   |                   |                      |                      |
        |                   |                   |     [if props provided]                     |
        |                   |                   |                      |---setProps---------->|
        |                   |                   |                      |   (by priority)      |                      |
        |                   |                   |                      |                      |
        |                   |                   |     [after batch - change detection]        |
        |                   |                   |---DetectChanges()--->|                      |
        |                   |                   |   (tracker)          | (self)               |
        |                   |                   |                      |                      |
        |                   |                   |     [for changed vars with watchers]        |
        |                   |                   |---send(update)------>|                      |
        |                   |                   |                      |---------------send-->|
        |                   |                   |                      |                      |
```

## Notes

- Property priority suffixes determine processing order
- Value and properties both optional in update
- Inactive variables suppress update notifications
- **Per-session change detection**: LuaBackend.DetectChanges() after batch processing
- **Per-session watchers**: Watchers map is in LuaBackend, not global
- Source of truth holder stores changes; relay-only forwards
