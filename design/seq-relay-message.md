# Sequence: Relay Message

**Source Spec:** protocol.md
**Use Case:** Bidirectional message forwarding between frontend and backend

## Participants

- Frontend: Browser client
- MessageRelay: Message coordinator
- ProtocolHandler: Message processor
- VariableStore: Variable storage (for unbound)
- Backend: External backend

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
        |                      |                      |     [backend processes and responds]       |
        |                      |                      |<-----------------------------------update---|
        |                      |                      |                      |                      |
        |                      |<--relayToFrontend----|                      |                      |
        |                      |                      |                      |                      |
        |<--update-------------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Unbound variables handled locally by UI server
- Bound variables forwarded to backend
- Backend processes and sends responses back
- Responses relayed to all watching frontends
- Message batching can combine multiple updates
