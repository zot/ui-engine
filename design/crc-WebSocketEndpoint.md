# WebSocketEndpoint

**Source Spec:** interfaces.md, deployment.md
**Requirements:** R40, R41, R159, R162, R163, R204

## Responsibilities

### Knows
- connections: Map of connection ID to WebSocket connection
- sessionBindings: Map of connection ID to session ID
- sessionSvc: Map of session ID to its ChanSvc executor, created and started on first use (R204)
- messageQueue: Outbound message queue per connection
- reconnectTokens: Map of session ID to reconnect token for reconnection validation

### Does
- accept: Accept new WebSocket connection
- close: Close connection and cleanup
- send: Send message to specific connection
- sendBatch: Send JSON array batch to connection
- broadcast: Send message to all connections in session
- receive: Handle incoming message (check for array batch, start timer before processing)
- bindToSession: Associate connection with session
- isConnected: Check connection status
- getSessionId: Return session for connection
- onDisconnect: Handle connection close
- isSessionReconnectable: Check if session exists and can be rejoined
- generateReconnectToken: Create token for validating reconnection to same session
- ExecuteInSessionPassive(sessionID, fn): queue fn on the session executor without waiting and without afterBatch (R159)
- RunAfterBatch(sessionID): run change detection and push now when a browser is connected; called on the executor (R163)
- cleanupSessionSvc(sessionID): stop the session's ChanSvc and forget it (R204)
- HasConnectionsForSession(sessionID): whether a browser is attached (R162)

## Collaborators

- Session: Connection belongs to session
- SessionManager: Queries session state during reconnection
- ProtocolHandler: Routes received messages
- MessageRelay: Coordinates message flow
- SharedWorker: Coordinates with other tabs
- Config: Logging delegate (connection events and errors)
- ChanSvc: one per session, serializes and orders session work

## Sequences

- seq-frontend-connect.md: WebSocket handshake and session binding
- seq-frontend-reconnect.md: Frontend reconnection to existing session
- seq-relay-message.md: Message routing
- seq-activate-tab.md: Tab activation via WebSocket
- seq-session-ref.md: posting through a session reference

## Notes

- Sessions can be reconnected to at any time before session timeout
- No separate connection timeout - session timeout handles cleanup
