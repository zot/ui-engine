# Sequence: Session Create Backend

**Source Spec:** main.md (UI Server Architecture)
**Use Case:** Creating a session with per-session backend and Lua VM initialization

## Participants

- Browser: User's web browser
- HTTPEndpoint: HTTP request handler
- SessionManager: Session lifecycle management
- Session: Frontend layer session
- Server: Main server (owns luaSessions map)
- LuaSession: Per-session Lua environment with isolated VM
- LuaBackend: Per-session backend for watch management
- luaTrackerAdapter: Routes variable operations to per-session tracker
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
        |                   |                   |                   |                   |--SetBackend(id, lb)---------------------------------------->|                   |
        |                   |                   |                   |                   |--SetLuaSession(id, ls)------------------------------------->|                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--SetVariableStore(adapter)---------->|                   |                   |
        |                   |                   |                   |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--luaSessions[id] = ls                |                   |                   |
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
- **Server Owns LuaSessions**: Server maintains `luaSessions map[string]*LuaSession`
- Each session gets its own LuaBackend with its own change-tracker.Tracker
- Variable IDs are scoped to the session (each session has its own variable 1)
- main.lua is responsible for creating the app variable via session:createAppVariable()
- The tracker is per-session, eliminating the global map key collision bug
- **luaTrackerAdapter**: Coordinates variable operations across per-session backends
- **Session Callbacks**: SessionManager.SetOnSessionCreated triggers Server.CreateLuaBackendForSession
