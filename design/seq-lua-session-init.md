# Sequence: Lua Session Initialization

**Source Spec:** interfaces.md, libraries.md
**Use Case:** Creating a Lua session when a new frontend session connects

## Participants

- Frontend: Browser frontend connecting
- HTTPEndpoint: HTTP request handler
- SessionManager: Session management
- Session: Frontend session instance
- Server: Main server (owns luaSessions map)
- LuaSession: Per-session Lua environment (with isolated VM)
- LuaBackend: Per-session backend for watch management
- luaTrackerAdapter: Routes to per-session tracker

## Sequence

```
     Frontend       HTTPEndpoint       SessionManager        Session             Server            LuaSession          LuaBackend      luaTrackerAdapter
        |                   |                   |                   |                   |                   |                   |                   |
        |---GET /---------->|                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |--createSession()->|                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |--new(id)--------->|                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |--onSessionCreated(vendedID, sess)--->|                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--NewRuntime()---->|                   |                   |
        |                   |                   |                   |                   |   [creates Lua    |                   |                   |
        |                   |                   |                   |                   |    VM state]      |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--NewLuaBackend()->|------------------>|                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |--SetBackend----->|<------------------|                   |                   |
        |                   |                   |                   |   (backend)      |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--SetBackend(id, lb)----------------->|----------------->|
        |                   |                   |                   |                   |--SetLuaSession(id, ls)-------------->|                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--SetVariableStore(adapter)---------->|                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--luaSessions[id] = ls                |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--CreateLuaSession(vendedID)--------->|                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |--CreateSession(id)----------------->|
        |                   |                   |                   |                   |                   |   [sets resolver] |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |--createSessionTable()               |
        |                   |                   |                   |                   |                   |--setGlobal("session")               |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |--loadMainLua()--->|                   |
        |                   |                   |                   |                   |                   |   [main.lua       |                   |
        |                   |                   |                   |                   |                   |    executes]      |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |   session:createAppVariable(value, props)
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |--CreateVariable(id, 0, obj, props)->|
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |<--variable 1------|-------------------|
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |<--luaSession------|                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |<--session---------|<------------------|                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |                   |<--sessionId-------|                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
        |<--302 /sessionId--|                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |
```

## Notes

- **Per-Session Isolation**: Each frontend session gets its own LuaSession with isolated Lua VM
- **Server Owns Sessions**: Server maintains `luaSessions map[string]*LuaSession`
- The server does NOT create variable 1 - main.lua creates it via session:createAppVariable()
- Executing main.lua serves as the notification that a new session has started
- The `session` global is available when main.lua executes
- Built-in watcher: `lua` property on variable 1 triggers dynamic code loading
- All Lua operations go through LuaSession's executor channel for thread safety
- **Session Callbacks**: SessionManager calls Server.CreateLuaBackendForSession on new session
- main.lua is responsible for:
  - Creating variable 1 (the app variable) with initial state
  - Defining presenter objects with methods for ui-action calls
