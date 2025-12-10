# Sequence: Destroy Variable

**Source Spec:** protocol.md
**Use Case:** Destroying a variable and its children

## Participants

- Sender: Frontend or backend initiating destroy
- ProtocolHandler: Message processor
- VariableStore: Variable storage
- WatchManager: Watch subscription manager
- MessageRelay: Message forwarding

## Sequence

```
     Sender            ProtocolHandler         VariableStore          WatchManager          MessageRelay
        |                      |                      |                      |                      |
        |---destroy(varId)---->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---getChildren------->|                      |                      |
        |                      |   (varId)            |                      |                      |
        |                      |                      |                      |                      |
        |                      |<--childIds-----------|                      |                      |
        |                      |                      |                      |                      |
        |                      |     [for each child, recursively]           |                      |
        |                      |---destroy(childId)-->|                      |                      |
        |                      |                      |                      |                      |
        |                      |---removeWatchers---->|                      |                      |
        |                      |   (varId)            |                      |                      |
        |                      |                      |---removeAll--------->|                      |
        |                      |                      |                      |                      |
        |                      |          [if bound and had watchers]        |                      |
        |                      |                      |---forwardUnwatch---->|                      |
        |                      |                      |                      |                      |
        |                      |---delete(varId)----->|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---delete()---------->|                      |
        |                      |                      |                      |                      |
        |                      |          [relay to other side]              |                      |
        |                      |----------------------------------------relay destroy------------>|
        |                      |                      |                      |                      |
```

## Notes

- Destruction is recursive - all children destroyed first
- Watchers automatically cleaned up before deletion
- Unwatch forwarded to backend if bound variable had observers
- Destroy relayed bidirectionally like other messages
- Source of truth holder removes from storage
