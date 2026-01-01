# SessionManager

**Source Spec:** interfaces.md, protocol.md

## Responsibilities

### Knows
- sessions: Map of session ID to Session
- urlPaths: Map of registered URL paths to presenter variables
- sessionTimeout: Duration for session inactivity cleanup
- luaSessionFactory: Factory for creating LuaSessions
- nextVendedID: Counter for sequential vended IDs (starts at 1)
- internalToVended: Map of internal session ID (UUID) to vended ID (string integer)
- vendedToInternal: Map of vended ID to internal session ID

### Does
- createSession: Generate new session ID, assign vended ID, create Session, trigger Lua session creation
- getSession: Retrieve session by ID (internal ID)
- destroySession: Clean up session, destroy Lua session, remove vended ID mappings, and all resources
- sessionExists: Check if session ID is valid
- registerUrlPath: Associate URL path with presenter for session
- resolveUrlPath: Find presenter for URL path
- generateSessionId: Create unique session identifier (internal UUID)
- cleanupInactiveSessions: Remove sessions with no activity past timeout
- getVendedID: Convert internal session ID to vended ID string
- getInternalID: Convert vended ID string to internal session ID

## Collaborators

- Session: Individual session instances
- LuaSession: Per-session Lua environment (created when frontend sessions are created, creates variable 1)
- Router: URL path registration
- Config: Provides session timeout setting

## Sequences

- seq-create-session.md: Full session creation (including Lua session)
- seq-navigate-url.md: URL path resolution
- seq-mcp-create-session.md: MCP session creation
- seq-frontend-reconnect.md: Reconnection to existing session

## Notes

- Sessions remain valid until session timeout expires (default 24h)
- Frontend can reconnect to any existing session at any time
- No separate connection timeout - session timeout handles all cleanup
- **Session-based Lua**: Each frontend session gets a corresponding LuaSession (when Lua enabled)

**Vended Session IDs (see protocol.md):**
- Internal session IDs are UUIDs (32 hex chars) for internal tracking and URL paths
- Vended IDs are sequential integers ("1", "2", "3"...) for backend communication
- Backend (Lua/external) only sees vended IDs, saving bandwidth vs full UUIDs
- SessionManager maintains bidirectional mapping between internal and vended IDs

**Backend Modes (see interfaces.md):**
- **Embedded Lua only**: LuaSession's main.lua creates variable 1
- **Connected backend only**: Backend creates variable 1 when it receives first session batch
- **Hybrid**: Both can create variables; developer decides where variable 1 is created
