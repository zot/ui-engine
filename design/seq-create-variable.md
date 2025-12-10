# Sequence: Create Variable

**Source Spec:** protocol.md
**Use Case:** Creating a new variable in the protocol

## Participants

- Sender: Frontend or backend initiating create
- ProtocolHandler: Message processor
- VariableStore: Variable storage
- WatchManager: Watch subscription manager
- MessageRelay: Message forwarding

## Sequence

```
     Sender            ProtocolHandler         VariableStore          WatchManager          MessageRelay
        |                      |                      |                      |                      |
        |--create(parent,val,--|                      |                      |                      |
        |   props,nowatch,     |                      |                      |                      |
        |   unbound)---------->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---parseProps-------->|                      |                      |
        |                      |   (extract :high/    |                      |                      |
        |                      |    :med/:low)        |                      |                      |
        |                      |                      |                      |                      |
        |                      |---create(parent,---->|                      |                      |
        |                      |    val,props,unbound)|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---allocateId()------>|                      |
        |                      |                      |                      |                      |
        |                      |                      |<--newId--------------|                      |
        |                      |                      |                      |                      |
        |                      |                      |---processProps------>|                      |
        |                      |                      |   (by priority)      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---store()----------->|                      |
        |                      |                      |                      |                      |
        |                      |<--variable-----------|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if not nowatch]                   |                      |
        |                      |---watch(varId)------>|                      |                      |
        |                      |                      |---addWatch---------->|                      |
        |                      |                      |                      |                      |
        |                      |          [if bound and tally 0->1]          |                      |
        |                      |                      |                      |---relay watch------->|
        |                      |                      |                      |                      |
        |                      |          [relay to other side]              |                      |
        |                      |----------------------------------------relay create-------------->|
        |                      |                      |                      |                      |
```

## Notes

- Properties with :high/:med/:low suffixes processed in priority order
- `create` property in props causes object instantiation
- `unbound` flag determines UI server vs backend source of truth
- `nowatch` flag skips automatic watch subscription
- Watch forwarding uses tally to avoid duplicate backend notifications
