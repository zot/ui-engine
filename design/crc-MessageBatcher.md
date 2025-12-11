# MessageBatcher

**Source Spec:** protocol.md

## Responsibilities

### Knows
- pendingChanges: Map of varId to queued value/property changes with priorities
- valuePriorities: Map of varId to value priority (high/medium/low)
- propertyPriorities: Map of varId to property priorities
- defaultPriority: Default priority for values (medium)
- sessionId: Session ID for session-wrapped batches (server-to-Lua communication)

### Does
- queueValue: Queue value change with priority
- queueProperty: Queue property change with priority (parsed from :suffix)
- parsePropertyPriority: Extract :high/:med/:low suffix from property name
- buildBatch: Create ordered JSON array from pending changes
- buildSessionBatch: Create session-wrapped batch {"session": id, "messages": [...]}
- separateByPriority: Group changes into high/medium/low buckets
- createUpdateMessage: Build update message for varId with value and/or properties
- flush: Build and return batch, clearing pending state
- flushWithSession: Build session-wrapped batch, clearing pending state
- isEmpty: Check if any changes are pending
- setSessionId: Set session ID for session-wrapped batches

## Collaborators

- MessageRelay: Uses batcher for outgoing messages
- ProtocolHandler: Queues changes via batcher
- ViewdefStore: Queues viewdef updates with :high priority
- LuaSession: Receives session-wrapped batches (when Lua enabled)
- BackendSocket: Receives session-wrapped batches (when backend connected)

## Sequences

- seq-viewdef-delivery.md: Priority-based viewdef batching
- seq-relay-message.md: Batched message sending

## Notes

**Session-based batching:**
- All batches to backends (Lua or external) include session wrapper
- Format: `{"session": "abc123", "messages": [...]}`
- Allows routing to correct session (LuaSession or backend session)
- Both LuaRuntime and BackendSocket use same session-wrapped format
