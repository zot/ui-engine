# ProtocolHandler

**Source Spec:** protocol.md, deployment.md, interfaces.md

## Responsibilities

### Knows
- unboundMode: Whether UI server is source of truth
- luaEnabled: Whether embedded Lua is active (--lua flag)
- backendConnected: Whether external backend is connected

### Does
- handleCreate: Process create(parentId, value, properties, nowatch?, unbound?) message
- handleDestroy: Process destroy(varId) message
- handleUpdate: Process update(varId, value?, properties?) message
- handleWatch: Process watch(varId) message
- handleUnwatch: Process unwatch(varId) message
- handleGet: Process get([varId, ...]) message (server-only)
- handleGetObjects: Process getObjects([objId, ...]) message (server-only)
- sendError: Send error(varId, code, description) to client (code is one-word like 'path-failure', 'not-found')
- relayToLua: Forward message to Lua session for processing
- relayToBackend: Forward message to connected backend via BackendSocket
- routeMessage: Determine whether message goes to Lua, backend, or both
- parsePropertyPriority: Extract :high/:med/:low suffixes from property names
- processPropertiesByPriority: Handle properties in priority order
- handleBatch: Process JSON array of messages in order
- handleSessionBatch: Process batch with session ID wrapper {"session": "id", "messages": [...]}
- isBatch: Check if incoming message is array (batch) or object (single)
- isSessionBatch: Check if message has session wrapper format

## Collaborators

- VariableStore: Modifies variable state
- WatchManager: Manages subscriptions
- MessageRelay: Forwards messages
- MessageBatcher: Builds priority-ordered batches for outgoing messages
- WebSocketEndpoint: Receives messages from frontend
- BackendSocket: Forwards messages to connected backend (when connected)
- LuaRuntime: Routes messages to appropriate LuaSession (when Lua enabled)
- LuaSession: Processes messages for session
- HTTPEndpoint: Receives messages via REST/CLI
- Config: Logging delegate (protocol messages and errors)

## Sequences

- seq-create-variable.md: Handling create message
- seq-update-variable.md: Handling update message
- seq-watch-variable.md: Handling watch/unwatch
- seq-relay-message.md: Message forwarding flow
- seq-viewdef-delivery.md: Batched viewdef delivery
- seq-lua-action-dispatch.md: Routing actions to Lua

## Notes

**Backend Modes (see interfaces.md):**
- **Embedded Lua only** (`--lua`, no backend): Route to LuaSession only
- **Connected backend only** (no `--lua`): Route to BackendSocket only
- **Hybrid** (`--lua` + backend): Route to both; developer controls behavior

**Message Routing:**
- Frontend messages (ui-action, etc.) routed based on backend mode
- In hybrid mode, both Lua and backend receive messages
- Variable 1 can be created by either Lua or backend (whoever creates it first wins)

**Session-based batching:**
- Protocol batches include session ID: `{"session": "abc123", "messages": [...]}`
- Session ID allows routing to correct LuaSession or backend session
