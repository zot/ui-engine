# Sequence: Bootstrap

**Source Spec:** viewdefs.md, interfaces.md
**Use Case:** Frontend app initialization and viewdef loading

## Participants

- Browser: User's web browser
- FrontendApp: Main frontend application
- SharedWorker: Coordinates tab connections
- WebSocketEndpoint: Server WebSocket handler
- VariableStore: Variable storage
- ViewdefStore: Viewdef storage

## Sequence

```
     Browser              FrontendApp           SharedWorker         WebSocketEndpoint      VariableStore        ViewdefStore
        |                      |                      |                      |                      |                      |
        |----navigate(URL)--->|                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---connect()--------->|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |---WebSocket open---->|                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |<--connection ack-----|                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |<--setMainTab---------|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---watch(1)---------->|                      |                      |                      |
        |                      |                      |---watch(1)---------->|                      |                      |
        |                      |                      |                      |---get(1)------------>|                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |<--variable 1---------|                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |---getViewdefs------->|                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |<--viewdefs map-------|                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |<--update(1,viewdefs)-|                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |<--update(1,viewdefs)-|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---storeViewdefs()--->|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---render()-----------|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |<---display view------|                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |
```

## Notes

- Frontend immediately watches variable 1 on connection
- Variable 1 contains root app object and viewdefs property
- Viewdefs are stored locally in frontend after receipt
- Render triggers after viewdefs are available
- SharedWorker designates first connecting tab as main tab
