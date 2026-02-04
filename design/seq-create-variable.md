# Sequence: Create Variable

**Source Spec:** protocol.md
**Requirements:** R43, R44, R45, R46, R47, R48
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
        |--create(id,------>|                   |                      |                      |
        |   parent,val,     |                   |                      |                      |
        |   props,nowatch,  |                   |                      |                      |
        |   unbound)        |                   |                      |                      |
        |                   |                   |                      |                      |
        | (id vended by     |                   |                      |                      |
        |  sender: frontend |                   |                      |                      |
        |  uses 2+, server  |                   |                      |                      |
        |  uses 1 for root  |                   |                      |                      |
        |  and -1- for rest)|                   |                      |                      |
        |                   |                   |                      |                      |
        |                   |--HandleMessage--->|                      |                      |
        |                   |                   |                      |                      |
        |                   |                   |---parseProps-------->|                      |
        |                   |                   |   (extract :high/    | (self)               |
        |                   |                   |    :med/:low)        |                      |
        |                   |                   |                      |                      |
        |                   |                   |---createWithId(id,-->|                      |
        |                   |                   |    parent,val,props, |                      |
        |                   |                   |    unbound)          |                      |
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

- **ID Vending**: Sender provides the variable ID (no server response needed)
  - Frontend vends IDs starting from 2 (incrementing)
  - Server vends ID 1 for root, then -1, -2, ... for other server-created variables
- Properties with :high/:med/:low suffixes processed in priority order
- `create` property in props causes object instantiation
- `unbound` flag determines UI server vs backend source of truth
- `nowatch` flag skips automatic watch subscription
- **Per-session watch management**: Watchers map and tally are in LuaBackend
- Watch registration with change-tracker occurs on tally 0->1 transition
- **No createResponse**: This is a push-only operation (no acknowledgment)
