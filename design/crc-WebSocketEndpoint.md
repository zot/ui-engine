# WebSocketEndpoint

**Source Spec:** interfaces.md, deployment.md

## Responsibilities

### Knows
- connections: Map of connection ID to WebSocket connection
- sessionBindings: Map of connection ID to session ID
- messageQueue: Outbound message queue per connection
- reconnectTokens: Map of session ID to reconnect token for reconnection validation

### Does
- accept: Accept new WebSocket connection
- close: Close connection and cleanup
- send: Send message to specific connection
- broadcast: Send message to all connections in session
- receive: Handle incoming message
- bindToSession: Associate connection with session
- isConnected: Check connection status
- getSessionId: Return session for connection
- onDisconnect: Handle connection close
- isSessionReconnectable: Check if session exists and can be rejoined
- generateReconnectToken: Create token for validating reconnection to same session

## Collaborators

- Session: Connection belongs to session
- SessionManager: Queries session state during reconnection
- ProtocolHandler: Routes received messages
- MessageRelay: Coordinates message flow
- SharedWorker: Coordinates with other tabs
- Config: Logging delegate (connection events and errors)

## Sequences

- seq-frontend-connect.md: WebSocket handshake and session binding
- seq-frontend-reconnect.md: Frontend reconnection to existing session
- seq-relay-message.md: Message routing
- seq-activate-tab.md: Tab activation via WebSocket

## Notes

- Sessions can be reconnected to at any time before session timeout
- No separate connection timeout - session timeout handles cleanup
