# Sequence: Backend Connect

**Source Spec:** libraries.md
**Use Case:** External backend connecting to UI server

## Participants

- Backend: External backend program
- BackendConnection: Connection manager
- WebSocketEndpoint: Server WebSocket handler
- SessionManager: Session management
- VariableStore: Variable storage

## Sequence

```
     Backend          BackendConnection      WebSocketEndpoint      SessionManager         VariableStore
        |                      |                      |                      |                      |
        |---connect(url,------>|                      |                      |                      |
        |    rootValue)        |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---WebSocket open---->|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getSession-------->|                      |
        |                      |                      |   (from URL)         |                      |
        |                      |                      |                      |                      |
        |                      |                      |<--session------------|                      |
        |                      |                      |                      |                      |
        |                      |<--connected----------|                      |                      |
        |                      |                      |                      |                      |
        |                      |---update(1,--------->|                      |                      |
        |                      |    rootValue)        |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---update(1)--------->|                      |
        |                      |                      |                      |---setValue---------->|
        |                      |                      |                      |                      |
        |                      |                      |---notifyWatchers---->|                      |
        |                      |                      |                      |                      |
        |                      |---setCloseHook------>|                      |                      |
        |                      |                      |                      |                      |
        |<--connected----------|                      |                      |                      |
        |                      |                      |                      |                      |
        |     [backend receives messages]             |                      |                      |
        |<--watch/update/etc---|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Backend provides root value for variable 1
- Root value should bind to currentPage() for SPA apps
- Close hook invoked when connection terminates
- Backend receives protocol messages after connection
- Backend can create bound variables for external data
