# MessageRelay

**Source Spec:** protocol.md, deployment.md

## Responsibilities

### Knows
- frontendConnections: Map of session to frontend connections
- backendConnections: Map of session to backend connections
- messageBuffer: Buffer for batching messages
- pendingQueues: Map of connection to PendingResponseQueue

### Does
- relayToFrontend: Forward message from backend to frontend
- relayToBackend: Forward message from frontend to backend
- shouldRelay: Determine if message should be forwarded
- filterForUnbound: Handle unbound variable messages locally
- batchMessages: Combine multiple updates for efficiency
- flushBatch: Send batched messages
- enqueuePending: Add push message (update, error, destroy) to pending queue

## Collaborators

- WebSocketEndpoint: Frontend connections
- BackendConnection: Backend connections
- ProtocolHandler: Message processing
- VariableStore: Unbound variable handling
- PendingResponseQueue: Accumulates push messages for polling clients

## Sequences

- seq-relay-message.md: Full relay flow
- seq-update-variable.md: Update message relay
- seq-watch-variable.md: Watch message relay with tallying
- seq-poll-pending.md: Pending message accumulation
