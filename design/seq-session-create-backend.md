# Sequence: Session Create Backend

**Source Spec:** main.md (UI Server Architecture)
**Use Case:** Creating a session with per-session backend initialization

## Participants

- Browser: User's web browser
- HTTPEndpoint: HTTP request handler
- SessionManager: Session lifecycle management
- Session: Frontend layer session
- LuaBackend: Per-session Lua backend
- Tracker: change-tracker.Tracker instance

## Sequence

```
     Browser          HTTPEndpoint        SessionManager          Session            LuaBackend            Tracker
        |                   |                   |                   |                   |                   |
        |---GET /---------->|                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |
        |                   |--createSession()->|                   |                   |                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |--new(id)--------->|                   |                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |--new(session)--->|                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--New(resolver)--->|
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |<--tracker---------|
        |                   |                   |                   |                   |   (per-session)   |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |--loadMainLua()--->|
        |                   |                   |                   |                   |   (self)          |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |                   |   [main.lua       |
        |                   |                   |                   |                   |    creates var 1] |
        |                   |                   |                   |                   |                   |
        |                   |                   |                   |<--backend---------|                   |
        |                   |                   |                   |                   |                   |
        |                   |                   |<--session---------|                   |                   |
        |                   |                   |                   |                   |                   |
        |                   |<--sessionId-------|                   |                   |                   |
        |                   |                   |                   |                   |                   |
        |<--302 /sessionId--|                   |                   |                   |                   |
        |                   |                   |                   |                   |                   |
```

## Notes

- Each session gets its own LuaBackend with its own change-tracker.Tracker
- Variable IDs are scoped to the session (each session has its own variable 1)
- main.lua is responsible for creating the app variable via session:createAppVariable()
- The tracker is per-session, eliminating the global map key collision bug
