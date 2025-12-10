# ProtocolHandler

**Source Spec:** protocol.md, deployment.md

## Responsibilities

### Knows
- boundConnections: Map of connection ID to backend connection
- unboundMode: Whether UI server is source of truth

### Does
- handleCreate: Process create(parentId, value, properties, nowatch?, unbound?) message
- handleDestroy: Process destroy(varId) message
- handleUpdate: Process update(varId, value?, properties?) message
- handleWatch: Process watch(varId) message
- handleUnwatch: Process unwatch(varId) message
- handleGet: Process get([varId, ...]) message (server-only)
- handleGetObjects: Process getObjects([objId, ...]) message (server-only)
- handlePoll: Process poll(wait?) message for pending responses
- sendError: Send error(varId, description) to client
- relayMessage: Forward message between frontend and backend
- parsePropertyPriority: Extract :high/:med/:low suffixes from property names
- processPropertiesByPriority: Handle properties in priority order

## Collaborators

- VariableStore: Modifies variable state
- WatchManager: Manages subscriptions
- MessageRelay: Forwards messages
- WebSocketEndpoint: Receives messages from frontend
- BackendConnection: Receives messages from backend
- PacketProtocol: Receives messages via packet protocol
- HTTPEndpoint: Receives messages via REST/CLI
- PendingResponseQueue: Accumulates push messages

## Sequences

- seq-create-variable.md: Handling create message
- seq-update-variable.md: Handling update message
- seq-watch-variable.md: Handling watch/unwatch
- seq-relay-message.md: Message forwarding flow
- seq-backend-socket-accept.md: Socket-based message handling
- seq-poll-pending.md: Polling for pending responses
