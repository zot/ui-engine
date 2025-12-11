# Sequence: Bootstrap

**Source Spec:** viewdefs.md, interfaces.md
**Use Case:** Frontend app initialization and viewdef loading

## Participants

- Browser: User's web browser
- FrontendApp: Main frontend application
- AppView: Root app view (ui-app element)
- SharedWorker: Coordinates tab connections
- WebSocketEndpoint: Server WebSocket handler
- VariableStore: Variable storage
- ViewdefStore: Viewdef storage
- View: Renders variable with viewdef

## Sequence

```
     Browser            FrontendApp             AppView           SharedWorker       WebSocketEndpoint     ViewdefStore           View
        |                    |                    |                    |                    |                    |                    |
        |---navigate(URL)--->|                    |                    |                    |                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |---find(ui-app)---->|                    |                    |                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |<--element----------|                    |                    |                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |---create(AppView)--|                    |                    |                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |---connect()--------|---------------->---|                    |                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |                    |                    |---WebSocket open-->|                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |                    |                    |<--connection ack---|                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |<--setMainTab-------|---------------<----|                    |                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |---initialize()---->|                    |                    |                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |                    |---watch(1)---------|---------------->---|                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |                    |                    |<--update(1,type,viewdefs)---------------|                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |                    |<--update(1)--------|---------------<----|                    |                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |                    |---storeViewdefs()--|-----------------|----------------->-----|                    |
        |                    |                    |                    |                    |                    |                    |
        |                    |                    |---render()---------|-----------------|----------------->-----|---------------->---|
        |                    |                    |                    |                    |                    |                    |
        |                    |                    |                    |                    |                    |---cloneTemplate--->|
        |                    |                    |                    |                    |                    |                    |
        |                    |                    |                    |                    |                    |<--true (rendered)--|
        |                    |                    |                    |                    |                    |                    |
        |<--display view-----|---------------<----|                    |                    |                    |                    |
        |                    |                    |                    |                    |                    |                    |
```

## Notes

- FrontendApp finds ui-app element on DOMContentLoaded and creates AppView
- AppView watches variable 1 (the root app variable) on initialization
- Variable 1 contains root app object with type property and viewdefs
- AppView uses View to render when variable 1 has type property and viewdef exists
- Viewdefs are stored locally in frontend ViewdefStore after receipt
- SharedWorker designates first connecting tab as main tab
