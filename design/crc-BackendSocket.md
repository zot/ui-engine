# BackendSocket

**Source Spec:** deployment.md, interfaces.md

## Responsibilities

### Knows
- socketPath: Path to socket (POSIX: Unix domain, Windows: named pipe)
- listener: Active socket listener
- connection: Active backend connection (single connection)
- sessionBatchers: Map of session ID to outbound batchers

### Does
- listen: Start listening on platform-appropriate socket
- accept: Accept incoming backend connection
- getDefaultPath: Return platform-specific default path (/tmp/ui.sock or \\.\pipe\ui)
- close: Close listener and connection
- sendToBackend: Send session-wrapped batch to backend
- handleIncoming: Process incoming session-wrapped batches from backend
- routeToSession: Route incoming batch to appropriate session for processing

## Collaborators

- Config: Logging delegate (socket events and errors) and provides socket path
- SessionManager: Creates sessions when backend sends first message for a session
- ProtocolHandler: Processes messages within batches
- MessageBatcher: Builds session-wrapped batches for outgoing messages

## Sequences

- seq-server-startup.md: Socket initialization
- seq-backend-socket-accept.md: Connection acceptance

## Notes

**Session-Based Batching:**
- All messages between UI server and backend are wrapped: `{"session": "abc123", "messages": [...]}`
- When backend sends batch with new session ID, SessionManager creates the session
- Backend is responsible for creating variable 1 (in connected backend or hybrid modes)
- Outbound batches are collected per-session and sent periodically

**Backend Modes (see interfaces.md):**
- **Embedded Lua only**: BackendSocket not used (no connected backend)
- **Connected backend only**: Backend creates variable 1 and handles all logic
- **Hybrid**: Both active; developer chooses where to create variable 1
