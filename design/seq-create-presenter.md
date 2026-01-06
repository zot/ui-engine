# Sequence: Create Presenter

**Source Spec:** main.md
**Use Case:** Creating a new presenter instance

## Participants

- Creator: Backend or MCP tool creating presenter
- ProtocolHandler: Message processor
- VariableStore: Variable storage
- ViewdefStore: Viewdef storage
- LuaBackend: Per-session watcher management

## Sequence

```
     Creator           ProtocolHandler         VariableStore          ViewdefStore           LuaBackend
        |                      |                      |                      |                      |
        |---create(parent,---->|                      |                      |                      |
        |    data,{type:T})    |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---create()---------->|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---allocateId()------>|                      |
        |                      |                      |                      |                      |
        |                      |                      |---setType(T)-------->|                      |
        |                      |                      |                      |                      |
        |                      |                      |---store()----------->|                      |
        |                      |                      |                      |                      |
        |                      |<--variable-----------|                      |                      |
        |                      |                      |                      |                      |
        |                      |---hasViewdef(T.DEF)->|                      |                      |
        |                      |                      |---has(T.DEFAULT)?--->|                      |
        |                      |                      |                      |                      |
        |                      |          [if viewdef exists]                |                      |
        |                      |                      |<--true---------------|                      |
        |                      |                      |                      |                      |
        |                      |---queueViewdef------>|                      |                      |
        |                      |   (for var 1 update) |                      |                      |
        |                      |                      |---batchUpdate------->|                      |
        |                      |                      |                      |                      |
        |                      |---notifyWatchers---->|                      |                      |
        |                      |                      |                      |---notify------------>|
        |                      |                      |                      |                      |
        |<--varId--------------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Presenter data stored as variable value
- Type property set to presenter type name
- Viewdefs for type queued for delivery via variable 1
- Watchers notified of new presenter
- Presenter can be child of another variable (tree structure)
