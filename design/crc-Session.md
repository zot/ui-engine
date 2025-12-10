# Session

**Source Spec:** main.md, interfaces.md

## Responsibilities

### Knows
- id: Unique session identifier (URL path component)
- appVariable: Reference to variable 1 (root app variable)
- connections: List of connected frontend/backend connections
- createdAt: Session creation timestamp
- lastActivity: Last activity timestamp
- disconnectedAt: Timestamp when last frontend disconnected (nil if connected)
- connectionTimeoutTimer: Timer for connection timeout grace period

### Does
- getId: Return session ID
- getAppVariable: Return root variable reference
- addConnection: Register new connection
- removeConnection: Unregister connection
- isActive: Check if session has any connections
- getConnectionCount: Return number of active connections
- touch: Update lastActivity timestamp
- startConnectionTimeout: Begin grace period countdown after frontend disconnect
- cancelConnectionTimeout: Cancel grace period when frontend reconnects
- isInGracePeriod: Check if session is disconnected but within grace period
- getGraceTimeRemaining: Return time left in grace period (for diagnostics)

## Collaborators

- SessionManager: Creates and destroys sessions, notified when grace period expires
- AppPresenter: Root presenter for session
- Variable: Variable 1 is session root
- WebSocketEndpoint: Frontend connections, triggers grace period on disconnect
- BackendConnection: Backend connections
- Config: Provides connection timeout duration

## Sequences

- seq-create-session.md: Session creation flow
- seq-frontend-connect.md: Frontend connecting to session
- seq-backend-connect.md: Backend connecting to session
- seq-frontend-reconnect.md: Frontend reconnecting within grace period
