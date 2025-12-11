# Sequence: Lua Session Initialization

**Source Spec:** interfaces.md, libraries.md
**Use Case:** Creating a Lua session when a new frontend session connects

## Participants

- Frontend: Browser frontend connecting
- WebSocketEndpoint: WebSocket handler
- SessionManager: Session management
- Session: Frontend session instance
- LuaRuntime: Lua VM manager
- LuaSession: Per-session Lua environment
- VariableStore: Variable storage
- WatchManager: Property watching

## Sequence

```
     Frontend       WebSocketEndpoint    SessionManager        Session           LuaRuntime          LuaSession         VariableStore       WatchManager
        |                   |                   |                   |                   |                   |                   |                   |
        |---connect-------->|                   |                   |                   |                   |                   |                   |
        |   (new session)   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |---createSession-->|                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |---create()------->|                   |                   |                   |                   |
        |                   |                   |   (sessionId)     |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |---createLuaSession(sessionId)-------->|                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |---execute-------->|                   |                   |
        |                   |                   |                   |                   |   (create)        |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |---new()---------->|                   |
        |                   |                   |                   |                   |                   |   [create Lua     |                   |
        |                   |                   |                   |                   |                   |    VM state]      |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |---setGlobal------>|                   |
        |                   |                   |                   |                   |                   |   ("session")     |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |---loadFile------->|                   |
        |                   |                   |                   |                   |                   |   ("main.lua")    |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |   [main.lua       |                   |
        |                   |                   |                   |                   |                   |    executes]      |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |   session:createAppVariable(value, props)
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |---create(1)------>|                   |
        |                   |                   |                   |                   |                   |   (value, props)  |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |<--variable 1------|                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |---watchProperty-->|----------------->|
        |                   |                   |                   |                   |                   |   (1, "lua")      |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |<--subscription---|
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |<--luaSession------|                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |<--luaSession------|                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |---setLuaSession-->|                   |                   |                   |                   |
        |                   |                   |   (luaSession)    |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |<--session---------|                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |<--connection ack--|                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
```

## Notes

- Each frontend session gets exactly one corresponding LuaSession
- The server does NOT create variable 1 - main.lua creates it via session:createAppVariable()
- Executing main.lua serves as the notification that a new session has started
- The `session` global is available when main.lua executes
- Built-in watcher: `lua` property on variable 1 triggers dynamic code loading
- All Lua operations go through LuaRuntime's executor channel for thread safety
- main.lua is responsible for:
  - Creating variable 1 (the app variable) with initial state
  - Defining presenter objects with methods for ui-action calls
