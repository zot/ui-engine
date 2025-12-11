# Sequence: Lua Handle Action

**Source Spec:** interfaces.md, viewdefs.md
**Use Case:** Frontend ui-action triggers path-based method call on presenter

## Participants

- Frontend: Browser frontend
- WebSocketEndpoint: WebSocket handler
- ProtocolHandler: Message processor
- LuaRuntime: Lua runtime manager
- LuaSession: Per-session Lua environment
- PathNavigator: Path resolution
- PresenterObject: Lua presenter table
- VariableStore: Variable storage

## Sequence

```
     Frontend       WebSocketEndpoint    ProtocolHandler       LuaRuntime          LuaSession        PathNavigator      PresenterObject      VariableStore
        |                   |                   |                   |                   |                   |                   |                   |
        |---update--------->|                   |                   |                   |                   |                   |                   |
        |   (varId,         |                   |                   |                   |                   |                   |                   |
        |    props:{        |                   |                   |                   |                   |                   |                   |
        |      path:        |                   |                   |                   |                   |                   |                   |
        |      "presenter.  |                   |                   |                   |                   |                   |                   |
        |       save()"})   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |---handleMessage-->|                   |                   |                   |                   |                   |
        |                   |   (sessionId,     |                   |                   |                   |                   |                   |
        |                   |    update)        |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |---getLuaSession-->|                   |                   |                   |                   |
        |                   |                   |   (sessionId)     |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |<--luaSession------|                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |---execute-------->|                   |                   |                   |                   |
        |                   |                   |   (dispatch       |                   |                   |                   |                   |
        |                   |                   |    action)        |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |---dispatchAction->|                   |                   |                   |
        |                   |                   |                   |   (varId, path,   |                   |                   |                   |
        |                   |                   |                   |    value?)        |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |---getVariable---->|                   |                   |
        |                   |                   |                   |                   |   (varId)         |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |---resolve-------->|                   |                   |
        |                   |                   |                   |                   |   (varValue,      |                   |                   |
        |                   |                   |                   |                   |    "presenter")   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |<--presenterObj----|                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |---callMethod(-----|----------------->|                   |
        |                   |                   |                   |                   |    "save",        |                   |                   |
        |                   |                   |                   |                   |    args?)         |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |---execute method  |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |---modify state--->|
        |                   |                   |                   |                   |                   |                   |   (var:update)    |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |<--result----------|------------------|                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |<--result----------|                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |<--result----------|                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |---notifyChanges-->|                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |<--update(s)-------|                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
```

## Notes

- **Path-based dispatch**: ui-action uses path syntax to resolve target method
- Path format: `object.method()` or `object.method(_)`
  - `method()` - call with no arguments
  - `method(_)` - call with the update message's value as the argument
- **Thread safety**: All Lua access goes through LuaRuntime's executor channel
- PathNavigator resolves the path to find the target object
- Method is called on the resolved presenter object
- Lua method can modify presenter state via variable updates
- Changes propagate back to frontend via normal update mechanism
- Error handling returns error message to frontend
