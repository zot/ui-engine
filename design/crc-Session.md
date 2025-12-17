# Session

**Source Spec:** main.md (UI Server Architecture - Frontend Layer), interfaces.md

## Responsibilities

### Knows
- id: Unique session identifier (URL path component)
- backend: Backend instance (LuaBackend or ProxiedBackend)
- connections: List of connected frontend connections
- createdAt: Session creation timestamp
- lastActivity: Last activity timestamp

### Does
- getId: Return session ID
- getBackend: Return backend instance
- addConnection: Register new frontend connection
- removeConnection: Unregister frontend connection, call backend.UnwatchAll for connection
- isActive: Check if session has any connections
- getConnectionCount: Return number of active connections
- touch: Update lastActivity timestamp
- handleMessage: Delegate to backend.HandleMessage

## Collaborators

- SessionManager: Creates and destroys sessions based on timeout
- Backend: Handles all protocol messages (LuaBackend or ProxiedBackend)
- WebSocketEndpoint: Frontend connections

## Sequences

- seq-session-create-backend.md: Session creation with backend initialization
- seq-frontend-connect.md: Frontend connecting to session
- seq-frontend-reconnect.md: Frontend reconnecting to existing session

## Notes

- **Lightweight frontend layer**: Session is part of the frontend layer; it routes messages to backend
- **No direct variable access**: Session does not manage variables - that is backend's responsibility
- **Backend delegation**: All watch/unwatch/protocol messages delegated to backend
- **Connection cleanup**: When connection removed, calls backend.UnwatchAll to clean up watches
- Sessions can be reconnected to at any time before session timeout expires
- Session timeout (default 24h) controls when inactive sessions are cleaned up
- Session ID is embedded in URL path for bookmarking and sharing
