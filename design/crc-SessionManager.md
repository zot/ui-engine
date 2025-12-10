# SessionManager

**Source Spec:** interfaces.md

## Responsibilities

### Knows
- sessions: Map of session ID to Session
- urlPaths: Map of registered URL paths to presenter variables
- gracePeriodSessions: Set of session IDs currently in connection timeout grace period

### Does
- createSession: Generate new session ID and initialize session
- getSession: Retrieve session by ID (includes sessions in grace period)
- destroySession: Clean up session and all resources
- sessionExists: Check if session ID is valid (includes sessions in grace period)
- registerUrlPath: Associate URL path with presenter for session
- resolveUrlPath: Find presenter for URL path
- generateSessionId: Create unique session identifier
- cleanupInactiveSessions: Remove sessions with no activity
- onConnectionTimeout: Handle session grace period expiration (may destroy session)
- getSessionsInGracePeriod: Return list of sessions awaiting reconnection

## Collaborators

- Session: Individual session instances
- Router: URL path registration
- VariableStore: Creates root variable
- AppPresenter: Creates root app presenter
- Config: Provides connection timeout duration for grace period

## Sequences

- seq-create-session.md: Full session creation
- seq-navigate-url.md: URL path resolution
- seq-mcp-create-session.md: MCP session creation
- seq-frontend-reconnect.md: Reconnection within grace period
