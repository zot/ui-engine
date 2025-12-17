# Sequence: Create Session

**Source Spec:** interfaces.md, main.md (UI Server Architecture)
**Use Case:** Creating a new session when user accesses the site

## Participants

- Browser: User's web browser
- HTTPEndpoint: HTTP request handler
- SessionManager: Session lifecycle management
- Session: Frontend layer session
- LuaBackend: Per-session Lua backend

## Sequence

```
     Browser          HTTPEndpoint        SessionManager          Session            LuaBackend
        |                   |                   |                   |                   |
        |---GET /---------->|                   |                   |                   |
        |                   |                   |                   |                   |
        |                   |--createSession()->|                   |                   |
        |                   |                   |                   |                   |
        |                   |                   |--new(id)--------->|                   |
        |                   |                   |                   |                   |
        |                   |                   |                   |--new(session)--->|
        |                   |                   |                   |   [creates       |
        |                   |                   |                   |    per-session   |
        |                   |                   |                   |    tracker]      |
        |                   |                   |                   |                   |
        |                   |                   |                   |--loadMainLua()-->|
        |                   |                   |                   |                   | (self)
        |                   |                   |                   |                   |
        |                   |                   |                   |   [main.lua      |
        |                   |                   |                   |    creates var 1]|
        |                   |                   |                   |                   |
        |                   |                   |                   |<--backend--------|
        |                   |                   |                   |                   |
        |                   |                   |<--session---------|                   |
        |                   |                   |                   |                   |
        |                   |<--sessionId-------|                   |                   |
        |                   |                   |                   |                   |
        |<--302 /sessionId--|                   |                   |                   |
        |                   |                   |                   |                   |
        |---GET /sessionId->|                   |                   |                   |
        |                   |                   |                   |                   |
        |<--index.html------|                   |                   |                   |
        |                   |                   |                   |                   |
```

## Notes

- Root URL redirects to session-specific URL
- Session ID embedded in URL path
- Session creates LuaBackend which owns per-session change-tracker
- Variable 1 is created by main.lua, not by the server
- Session URL can be bookmarked for reconnection
- Frontend app bootstraps after receiving HTML
- See seq-session-create-backend.md for detailed backend initialization
