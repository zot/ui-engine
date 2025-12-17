# Sequence: Backend Watch

**Source Spec:** protocol.md (Watch tallying)
**Use Case:** Subscribing to variable changes through Backend interface

## Participants

- Frontend: Browser client watching variable
- Session: Frontend layer session
- LuaBackend: Per-session Lua backend
- Tracker: change-tracker.Tracker instance

## Sequence

```
     Frontend              Session            LuaBackend             Tracker
        |                      |                   |                   |
        |---watch(varId)------>|                   |                   |
        |                      |                   |                   |
        |                      |--Watch(connId,--->|                   |
        |                      |        varId)     |                   |
        |                      |                   |                   |
        |                      |                   |--incrementTally-->|
        |                      |                   |   (watchCounts)   | (self)
        |                      |                   |                   |
        |                      |                   |--addWatcher------>|
        |                      |                   |   (watchers map)  | (self)
        |                      |                   |                   |
        |                      |                   |   [if tally was 0]|
        |                      |                   |--Watch(varId)---->|
        |                      |                   |                   |
        |                      |                   |<--ok--------------|
        |                      |                   |                   |
        |                      |                   |--getValue-------->|
        |                      |                   |                   |
        |                      |                   |<--value-----------|
        |                      |                   |                   |
        |                      |<--ok--------------|                   |
        |                      |                   |                   |
        |<--update(varId,val)--|                   |                   |
        |                      |                   |                   |
```

## Unwatch Sequence

```
     Frontend              Session            LuaBackend             Tracker
        |                      |                   |                   |
        |---unwatch(varId)---->|                   |                   |
        |                      |                   |                   |
        |                      |--Unwatch(connId,->|                   |
        |                      |          varId)   |                   |
        |                      |                   |                   |
        |                      |                   |--decrementTally-->|
        |                      |                   |   (watchCounts)   | (self)
        |                      |                   |                   |
        |                      |                   |--removeWatcher--->|
        |                      |                   |   (watchers map)  | (self)
        |                      |                   |                   |
        |                      |                   |   [if tally now 0]|
        |                      |                   |--Unwatch(varId)-->|
        |                      |                   |                   |
        |                      |                   |<--ok--------------|
        |                      |                   |                   |
        |                      |<--ok--------------|                   |
        |                      |                   |                   |
```

## Notes

- watchCounts and watchers maps are per-LuaBackend (per-session), not global
- Variable IDs are only unique within a session, so per-session maps are required
- Watch immediately returns current value via update message
- Tally tracks number of observers per variable within the session
- When tally goes 0->1, variable is registered with change-tracker
- When tally goes 1->0, variable is unregistered from change-tracker
