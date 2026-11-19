# Sequence: Session Create Backend

**Source Spec:** main.md (UI Server Architecture)
**Use Case:** Creating a session with per-session backend and Lua VM initialization

## Participants

- Browser: User's web browser
- HTTPEndpoint: HTTP request handler
- SessionManager: Session lifecycle management
- Session: Frontend layer session
- Server: Main server (finds Lua sessions through the session registry)
- LuaSession: Per-session Lua environment with isolated VM
- LuaBackend: Per-session backend for watch management
- luaTrackerAdapter: Routes each variable operation to the named session's tracker, found through the registry
- Tracker: change-tracker.Tracker instance

## Sequence

```
     Browser          HTTPEndpoint        SessionManager          Session             Server            LuaSession          LuaBackend      luaTrackerAdapter       Tracker
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |---GET /---------->|                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |--createSession()->|                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |--new(id)--------->|                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |--onSessionCreated(vendedID, sess)--->|                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--NewRuntime(cfg,  |                   |                   |                   |
        |                   |                   |                   |                   |   luaDir, vdm)--->|                   |                   |                   |
        |                   |                   |                   |                   |   [new Lua state] |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--NewLuaBackend(cfg, vendedID, resolver)----------------->|                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |--New(resolver)--->|----------------->|
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |<--tracker---------|<-----------------|
        |                   |                   |                   |                   |                   |                   |   (per-session)   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |--SetBackend(lb)--|<------------------|-------------------|                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--SetVariableStore(adapter)---------->|                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--Session.attachLua(ls)               |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--defineExecAsync(ref): ui.execAsync (R250)                |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--CreateLuaSession(vendedID)--------->|                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |--createSessionTable()               |                   |
        |                   |                   |                   |                   |                   |--setGlobal("session")               |                   |
        |                   |                   |                   |                   |                   |--loadMainLua()--->|                   |                   |
        |                   |                   |                   |                   |                   |   [main.lua       |                   |                   |
        |                   |                   |                   |                   |                   |    creates var 1] |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |<--luaSession------|                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |<--session---------|<------------------|                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |<--sessionId-------|                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |<--302 /sessionId--|                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
```

## Notes

- **Per-Session Isolation**: Each session gets its own LuaSession with isolated Lua VM state
- **Session Owns Its Lua Session**: Server attaches each LuaSession to its Session and finds it through the session registry; it keeps no map of its own
- Each session gets its own LuaBackend with its own change-tracker.Tracker
- Variable IDs are scoped to the session (each session has its own variable 1)
- main.lua is responsible for creating the app variable via session:createAppVariable()
- The tracker is per-session, eliminating the global map key collision bug
- **luaTrackerAdapter**: Holds no per-session state; each variable operation names its session, and the adapter finds that session's backend through the registry (R239, R241)
- **Session Callbacks**: SessionManager.SetOnSessionCreated triggers Server.CreateLuaBackendForSession
