# Sequence: Update Variable

**Source Spec:** protocol.md
**Use Case:** Updating variable value and/or properties

## Participants

- Sender: Frontend or backend initiating update
- ProtocolHandler: Message processor
- VariableStore: Variable storage
- WatchManager: Watch subscription manager
- MessageRelay: Message forwarding

## Sequence

```
     Sender            ProtocolHandler         VariableStore          WatchManager          MessageRelay
        |                      |                      |                      |                      |
        |--update(varId,------>|                      |                      |                      |
        |   value?,props?)     |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---parseProps-------->|                      |                      |
        |                      |   (extract priority) |                      |                      |
        |                      |                      |                      |                      |
        |                      |---get(varId)-------->|                      |                      |
        |                      |                      |                      |                      |
        |                      |<--variable-----------|                      |                      |
        |                      |                      |                      |                      |
        |                      |     [if value provided]                     |                      |
        |                      |---setValue---------->|                      |                      |
        |                      |                      |                      |                      |
        |                      |     [if props provided]                     |                      |
        |                      |---setProps---------->|                      |                      |
        |                      |   (by priority)      |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---store()----------->|                      |
        |                      |                      |                      |                      |
        |                      |     [if variable is watched]                |                      |
        |                      |---getWatchers------->|                      |                      |
        |                      |                      |---getWatchers------->|                      |
        |                      |                      |                      |                      |
        |                      |                      |<--watcher list-------|                      |
        |                      |                      |                      |                      |
        |                      |     [if not inactive]                       |                      |
        |                      |---notifyWatchers---->|                      |                      |
        |                      |                      |                      |---send update------->|
        |                      |                      |                      |                      |
        |                      |          [relay to other side]              |                      |
        |                      |----------------------------------------relay update-------------->|
        |                      |                      |                      |                      |
```

## Notes

- Property priority suffixes determine processing order
- Value and properties both optional in update
- Inactive variables suppress update notifications
- Updates relayed bidirectionally (frontend <-> backend)
- Source of truth holder stores changes; relay-only forwards
