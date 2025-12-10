# Sequence: Lua Handle Action

**Source Spec:** interfaces.md
**Use Case:** Lua presenter handling user action

## Participants

- Frontend: Browser frontend
- ProtocolHandler: Message processor
- LuaRuntime: Lua VM manager
- LuaPresenterLogic: Presenter instance
- WatchManager: Variable watching

## Sequence

```
     Frontend          ProtocolHandler          LuaRuntime        LuaPresenterLogic        WatchManager
        |                      |                      |                      |                      |
        |---update(var,------->|                      |                      |                      |
        |    {action:method,   |                      |                      |                      |
        |     values:{...}})   |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---getPresenter------>|                      |                      |
        |                      |   (varId)            |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getObject--------->|                      |
        |                      |                      |   (objId)            |                      |
        |                      |                      |                      |                      |
        |                      |                      |<--luaPresenter-------|                      |
        |                      |                      |                      |                      |
        |                      |<--presenter----------|                      |                      |
        |                      |                      |                      |                      |
        |                      |---callMethod-------->|                      |                      |
        |                      |   (method, values)   |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---invoke()---------->|                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |---execute method---->|
        |                      |                      |                      |                      |
        |                      |                      |                      |---modify state------>|
        |                      |                      |                      |                      |
        |                      |                      |<--result-------------|                      |
        |                      |                      |                      |                      |
        |                      |                      |---notifyChange------>|                      |
        |                      |                      |                      |---sendUpdates------->|
        |                      |                      |                      |                      |
        |<--update(s)----------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Action triggers method call on Lua presenter
- Values from form/UI passed as arguments
- Lua method can modify presenter state
- Changes detected and sent as updates
- Multiple variables may update from single action
- Error handling returns error message to frontend
