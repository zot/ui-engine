# WebSocketEndpoint

**Source Spec:** interfaces.md, deployment.md

## Responsibilities

### Knows
- connections: Map of connection ID to WebSocket connection
- sessionBindings: Map of connection ID to session ID
- messageQueue: Outbound message queue per connection
- reconnectTokens: Map of session ID to reconnect token for grace period validation

### Does
- accept: Accept new WebSocket connection
- close: Close connection and cleanup, start grace period if last frontend
- send: Send message to specific connection
- broadcast: Send message to all connections in session
- receive: Handle incoming message
- bindToSession: Associate connection with session
- isConnected: Check connection status
- getSessionId: Return session for connection
- onDisconnect: Handle connection close, notify Session to start grace period
- onReconnect: Handle reconnection within grace period, restore session state
- isSessionReconnectable: Check if session is in grace period and can be rejoined
- generateReconnectToken: Create token for validating reconnection to same session

## Collaborators

- Session: Connection belongs to session, notified of disconnect/reconnect
- SessionManager: Queries session state during reconnection
- ProtocolHandler: Routes received messages
- MessageRelay: Coordinates message flow
- SharedWorker: Coordinates with other tabs
- Config: Provides connection timeout setting

## Sequences

- seq-frontend-connect.md: WebSocket handshake and session binding
- seq-frontend-reconnect.md: Frontend reconnection within grace period
- seq-relay-message.md: Message routing
- seq-activate-tab.md: Tab activation via WebSocket
