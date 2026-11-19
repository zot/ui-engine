# WebSocketEndpoint

**Source Spec:** interfaces.md, deployment.md
**Requirements:** R40, R41, R159, R162, R163, R228, R244, R245, R246, R247, R248

## Responsibilities

### Knows
- connections: Map of connection ID to WebSocket connection; the connection registry, an edge
- wsConn.session: the `*Session` the connection was opened on, set once at connect when the
  session exists, nil otherwise; never changed. The only connection-to-session record (R244, R245)
- messageQueue: Outbound message queue per connection

### Does
- accept: Accept new WebSocket connection
- close: Close connection and cleanup
- send: Send message to specific connection
- sendBatch: Send JSON array batch to connection
- broadcast: look the session up in the registry and send to the connections it lists
  (Session.ConnectionIDs); a destroyed session is no longer found, so nothing is sent (R247)
- receive: Handle incoming message (check for array batch, start timer before processing)
- register(connectionID, conn, sessionID): the binding step HandleWebSocket runs: look the
  session up once; record the wsConn with the session when it exists, nil otherwise, and
  call Session.AddConnection (R244)
- isConnected: Check connection status
- GetSessionIDForConnection(connectionID): the wsConn's session ID, or "" (R244)
- onDisconnect: Handle connection close; forget the wsConn and call its session's
  RemoveConnection, undoing both directions together (R244)
- isSessionReconnectable: Check if session exists and can be rejoined; no token is issued or
  checked (R248)
- executor(sessionID): the Session's executor through SessionManager, or nil when there is no such session; never creates one (R228)
- ExecuteInSession(sessionID, fn): no executor -> error "session not found"; else post and wait, returning ErrStopped when the session was destroyed meanwhile (R228)
- ExecuteInSessionAsync, ExecuteInSessionPassive: no executor -> drop (R228)
- readPump: hands each message to postMessage(wc, message), which posts it to the wsConn's
  session's executor and reports true; a connection with no session posts nothing and
  reports false (R246)
- ExecuteInSessionPassive(sessionID, fn): queue fn on the session executor without waiting and without afterBatch (R159)
- RunAfterBatch(sessionID): run change detection and push now when a browser is connected; called on the executor (R163)
- HasConnectionsForSession(sessionID): the session is in the registry and lists a connection;
  a destroyed session is no longer found, so it has none (R162, R247)

## Collaborators

- Session: Connection belongs to session; owns the executor this endpoint posts to
- SessionManager: the session registry, looked up once per connection at connect
- ProtocolHandler: Routes received messages
- MessageRelay: Coordinates message flow
- SharedWorker: Coordinates with other tabs
- Config: Logging delegate (connection events and errors)

## Sequences

- seq-frontend-connect.md: WebSocket handshake and session binding
- seq-frontend-reconnect.md: Frontend reconnection to existing session
- seq-relay-message.md: Message routing
- seq-activate-tab.md: Tab activation via WebSocket
- seq-session-ref.md: posting through a session reference

## Notes

- Sessions can be reconnected to at any time before session timeout
- No separate connection timeout - session timeout handles cleanup
