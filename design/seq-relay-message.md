# Sequence: Relay Message

**Source Spec:** protocol.md
**Use Case:** Bidirectional message forwarding between frontend and backend

## Participants

- Frontend: Browser client
- MessageRelay: Message coordinator
- ProtocolHandler: Message processor
- VariableStore: Variable storage (for unbound)
- Backend: External backend (LuaRuntime or connected backend)

## Sequence

```
     Frontend          MessageRelay         ProtocolHandler        VariableStore            Backend
        |                      |                      |                      |                      |
        |---update(varId,val)->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---shouldRelay?------>|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---isUnbound?-------->|                      |
        |                      |                      |                      |                      |
        |                      |          [if unbound - UI server is source of truth]              |
        |                      |                      |<--true---------------|                      |
        |                      |                      |                      |                      |
        |                      |                      |---store()----------->|                      |
        |                      |                      |                      |                      |
        |                      |                      |---notifyWatchers---->|                      |
        |                      |                      |                      |                      |
        |                      |<--handled locally----|                      |                      |
        |                      |                      |                      |                      |
        |                      |          [if bound - backend is source of truth]                  |
        |                      |                      |<--false--------------|                      |
        |                      |                      |                      |                      |
        |                      |---relayToBackend---->|                      |                      |
        |                      |                      |--------------------------------------msg--->|
        |                      |                      |                      |                      |
        |                      |                      |     [backend processes message]            |
        |                      |                      |<-----------------------------------result---|
        |                      |                      |                      |                      |
        |                      |                      |     [after message - trigger change detection]
        |                      |                      |---afterBatch-------->|--------------------->|
        |                      |                      |                      |                      |
        |                      |                      |                      |  [DetectChanges - see seq-backend-refresh.md]
        |                      |                      |                      |                      |
        |                      |                      |<------------------------------------updates-|
        |                      |                      |                      |                      |
        |                      |<--relayToFrontend----|                      |                      |
        |                      |                      |                      |                      |
        |<--update(s)----------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Unbound variables handled locally by UI server
- Bound variables forwarded to backend
- Backend processes messages (may modify internal state)
- **Automatic change detection**: After each message, afterBatch triggers DetectChanges
- DetectChanges uses change-tracker package (see seq-backend-refresh.md for details)
- Only changed values are sent as updates to watching frontends
- Message batching can combine multiple updates
