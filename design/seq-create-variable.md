# Sequence: Create Variable

**Source Spec:** protocol.md
**Use Case:** Creating a new variable in the protocol

## Participants

- Sender: Frontend or backend initiating create
- Session: Frontend layer session
- LuaBackend: Per-session backend (implements Backend interface)
- VariableStore: In-memory variable store
- Tracker: change-tracker.Tracker instance (per-session)

## Sequence

```
     Sender              Session            LuaBackend           VariableStore           Tracker
        |                   |                   |                      |                      |
        |--create(parent,-->|                   |                      |                      |
        |   val,props,      |                   |                      |                      |
        |   nowatch,unbound)|                   |                      |                      |
        |                   |                   |                      |                      |
        |                   |--HandleMessage--->|                      |                      |
        |                   |                   |                      |                      |
        |                   |                   |---parseProps-------->|                      |
        |                   |                   |   (extract :high/    | (self)               |
        |                   |                   |    :med/:low)        |                      |
        |                   |                   |                      |                      |
        |                   |                   |---create(parent,---->|                      |
        |                   |                   |    val,props,unbound)|                      |
        |                   |                   |                      |                      |
        |                   |                   |                      |---allocateId()------>|
        |                   |                   |                      |                      | (self)
        |                   |                   |                      |                      |
        |                   |                   |                      |<--newId--------------|
        |                   |                   |                      |                      |
        |                   |                   |                      |---processProps------>|
        |                   |                   |                      |   (by priority)      | (self)
        |                   |                   |                      |                      |
        |                   |                   |<--variable-----------|                      |
        |                   |                   |                      |                      |
        |                   |                   |          [if not nowatch]                   |
        |                   |                   |---addWatcher-------->|                      |
        |                   |                   |   (watchers map)     | (self)               |
        |                   |                   |                      |                      |
        |                   |                   |          [if tally 0->1]                    |
        |                   |                   |---Watch(varId)------>|                      |
        |                   |                   |                      |                      |
        |                   |                   |<--ok-----------------|                      |
        |                   |                   |                      |                      |
```

## Notes

- Properties with :high/:med/:low suffixes processed in priority order
- `create` property in props causes object instantiation
- `unbound` flag determines UI server vs backend source of truth
- `nowatch` flag skips automatic watch subscription
- **Per-session watch management**: Watchers map and tally are in LuaBackend
- Watch registration with change-tracker occurs on tally 0->1 transition
