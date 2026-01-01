# Sequence: Lua Handle Action

**Source Spec:** interfaces.md, viewdefs.md
**Use Case:** Frontend ui-action triggers path-based method call on presenter

## Participants

- Frontend: Browser frontend
- WebSocketEndpoint: WebSocket handler
- ProtocolHandler: Message processor
- LuaSession: Per-session Lua environment
- PathNavigator: Path resolution
- PresenterObject: Lua presenter table
- VariableStore: Variable storage

## Sequence

```
     Frontend       WebSocketEndpoint    ProtocolHandler          LuaSession        PathNavigator      PresenterObject
        |                   |                   |                   |                   |                   |                   |
        |---update--------->|                   |                   |                   |                   |                   |
        |   (varId,         |                   |                   |                   |                   |                   |
        |    props:{        |                   |                   |                   |                   |                   |
        |      path:        |                   |                   |                   |                   |                   |
        |      "presenter.  |                   |                   |                   |                   |                   |
        |       save()"})   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |
        |                   |---handleMessage-->|                   |                   |                   |
        |                   |   (sessionId,     |                   |                   |                   |
        |                   |    update)        |                   |                   |                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |---execute-------->|                   |                   |
        |                   |                   |   (dispatch       |                   |                   |
        |                   |                   |    action)        |                   |                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |---getApp--------->|                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |<--appObject-------|                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |---resolve-------->|                   |
        |                   |                   |                   |   (appObject,     |                   |
        |                   |                   |                   |    "presenter")   |                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |<--presenterObj----|-------------------|
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |---callMethod------|----------------->|
        |                   |                   |                   |   ("save", args?) |                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |  [modifies self  |
        |                   |                   |                   |                   |   directly]      |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |<--result----------|-------------------|
        |                   |                   |                   |                   |                   |
        |                   |                   |<--result----------|                   |                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |---afterBatch----->|                   |                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |---computeChanges->|                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |  [for each watched var:]             |
        |                   |                   |                   |  - serialize object to JSON          |
        |                   |                   |                   |  - compare to cached value           |
        |                   |                   |                   |  - if changed, queue update          |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |<--changedVars-----|                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |<--updates---------|                   |                   |
        |                   |                   |                   |                   |                   |
        |<--update(s)-------|                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |
```

## Notes

- **Path-based dispatch**: ui-action uses path syntax to resolve target method
- Path format: `object.method()` or `object.method(_)`
  - `method()` - call with no arguments
  - `method(_)` - call with the update message's value as the argument
- **Thread safety**: All Lua access goes through LuaSession's executor channel
- PathNavigator resolves the path starting from the app object (session:getApp())
- Method is called on the resolved presenter object
- **Direct object modification**: Lua methods modify `self` directly (no var:update calls)
- **Automatic change detection**: After message batch, framework:
  1. Iterates all watched variables
  2. Serializes each referenced Lua object to JSON
  3. Compares to cached previous value
  4. Sends update messages for changed variables
- Error handling returns error message to frontend
