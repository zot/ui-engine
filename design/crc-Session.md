# Session

**Source Spec:** main.md, interfaces.md

## Responsibilities

### Knows
- id: Unique session identifier (URL path component)
- appVariable: Reference to variable 1 (root app variable, created by Lua)
- connections: List of connected frontend connections
- createdAt: Session creation timestamp
- lastActivity: Last activity timestamp
- luaSession: Reference to corresponding LuaSession (when --lua enabled)

### Does
- getId: Return session ID
- getAppVariable: Return root variable reference (may be nil until Lua creates it)
- setAppVariable: Set root variable reference (called when Lua creates variable 1)
- addConnection: Register new frontend connection
- removeConnection: Unregister frontend connection
- isActive: Check if session has any connections
- getConnectionCount: Return number of active connections
- touch: Update lastActivity timestamp
- getLuaSession: Return corresponding LuaSession

## Collaborators

- SessionManager: Creates and destroys sessions based on timeout
- LuaSession: Corresponding Lua session (creates variable 1)
- Variable: Variable 1 is session root (created by Lua, not server)
- WebSocketEndpoint: Frontend connections

## Sequences

- seq-create-session.md: Session creation flow (triggers Lua session creation)
- seq-frontend-connect.md: Frontend connecting to session
- seq-frontend-reconnect.md: Frontend reconnecting to existing session

## Notes

- Sessions can be reconnected to at any time before session timeout expires
- Session timeout (default 24h) controls when inactive sessions are cleaned up
- Session ID is embedded in URL path for bookmarking and sharing
- **Variable 1 creation**: main.lua creates variable 1 via session:createAppVariable() - the server does NOT create it
- **Embedded Lua only**: External backend sockets removed; all backend logic runs in embedded Lua
