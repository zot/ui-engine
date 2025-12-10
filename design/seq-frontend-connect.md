# Sequence: Frontend Connect

**Source Spec:** interfaces.md
**Use Case:** Browser tab establishing connection to UI server

## Participants

- Browser: User's browser tab
- FrontendApp: Frontend application
- SharedWorker: Tab coordination worker
- WebSocketEndpoint: Server WebSocket handler
- Session: Session instance

## Sequence

```
     Browser            FrontendApp           SharedWorker         WebSocketEndpoint          Session
        |                      |                      |                      |                      |
        |---load page--------->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---parseSessionId---->|                      |                      |
        |                      |   (from URL)         |                      |                      |
        |                      |                      |                      |                      |
        |                      |---connect()--------->|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |     [if first tab]   |                      |
        |                      |                      |---new WebSocket()--->|                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |---validateSession--->|
        |                      |                      |                      |                      |
        |                      |                      |                      |<--session valid------|
        |                      |                      |                      |                      |
        |                      |                      |<--connected----------|                      |
        |                      |                      |                      |                      |
        |                      |                      |---setMainTab-------->|                      |
        |                      |                      |                      |                      |
        |                      |<--isMainTab:true-----|                      |                      |
        |                      |                      |                      |                      |
        |                      |     [if not first tab]                      |                      |
        |                      |<--isMainTab:false----|                      |                      |
        |                      |                      |                      |                      |
        |                      |---watch(1)---------->|                      |                      |
        |                      |                      |---forward watch----->|                      |
        |                      |                      |                      |---addConnection----->|
        |                      |                      |                      |                      |
        |                      |                      |<--update(1)----------|                      |
        |                      |<--update(1)----------|                      |                      |
        |                      |                      |                      |                      |
        |<--render view--------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Session ID parsed from URL path
- SharedWorker coordinates multiple tabs
- First tab becomes main tab with WebSocket
- Other tabs relay through SharedWorker
- All tabs watch variable 1 to receive viewdefs
