# MessageRelay

**Source Spec:** protocol.md, deployment.md, interfaces.md

## Responsibilities

### Knows
- frontendConnections: Map of session to frontend connections
- luaEnabled: Whether embedded Lua is active
- backendConnected: Whether external backend is connected
- messageBuffer: Buffer for batching messages
- pendingQueues: Map of connection to PendingResponseQueue

### Does
- relayToFrontend: Forward message from backend to frontend
- relayToBackend: Forward message from frontend to backend (Lua or external)
- shouldRelay: Determine if message should be forwarded
- filterForUnbound: Handle unbound variable messages locally
- batchMessages: Combine multiple updates for efficiency
- flushBatch: Send batched messages
- enqueuePending: Add push message (update, error, destroy) to pending queue

## Collaborators

- WebSocketEndpoint: Frontend connections
- LuaSession: Per-session Lua environment (when embedded Lua enabled)
- BackendSocket: External backend connection (when connected)
- ProtocolHandler: Message processing
- VariableStore: Unbound variable handling
- PendingResponseQueue: Accumulates push messages for polling clients

## Sequences

- seq-relay-message.md: Full relay flow
- seq-viewdef-delivery.md: Priority-based viewdef delivery
- seq-update-variable.md: Update message relay
- seq-watch-variable.md: Watch message relay with tallying
- seq-poll-pending.md: Pending message accumulation

## Notes

**Backend Modes (see interfaces.md):**
- **Embedded Lua only**: relayToBackend routes to LuaSession
- **Connected backend only**: relayToBackend routes to BackendSocket
- **Hybrid**: relayToBackend routes to both (developer controls variable 1 ownership)
