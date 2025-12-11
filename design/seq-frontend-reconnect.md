# Sequence: Frontend Reconnect

**Source Spec:** interfaces.md
**Use Case:** Browser reconnecting to existing session after disconnect (e.g., page refresh, network interruption)

## Participants

- Browser: User's browser tab
- FrontendApp: Frontend application
- SharedWorker: Tab coordination worker
- WebSocketEndpoint: Server WebSocket handler
- Session: Session instance
- SessionManager: Session management

## Sequence

### Disconnect Flow

```
     Browser            FrontendApp           SharedWorker         WebSocketEndpoint          Session
        |                      |                      |                      |                      |
        |---page unload------->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---beforeunload------>|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---close WebSocket--->|                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |---onDisconnect------>|
        |                      |                      |                      |                      |
        |                      |                      |                      |---removeConnection-->|
        |                      |                      |                      |                      |
        |                      |                      |                      |     [session remains]|
        |                      |                      |                      |     [until timeout]  |
        |                      |                      |                      |                      |
```

### Reconnect Flow

```
     Browser            FrontendApp           SharedWorker         WebSocketEndpoint          Session          SessionManager
        |                      |                      |                      |                      |                      |
        |---load page--------->|                      |                      |                      |                      |
        |  (same session URL)  |                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---parseSessionId---->|                      |                      |                      |
        |                      |   (from URL)         |                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---connect()--------->|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |---new WebSocket()--->|                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |---sessionExists?----------------------------->|
        |                      |                      |                      |<--true-------------------------------------|
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |---addConnection----->|                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |<--connected----------|                      |                      |
        |                      |                      |   (session restored) |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |<--isMainTab:true-----|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---watch(1)---------->|                      |                      |                      |
        |                      |                      |---forward watch----->|                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |<--update(1)----------|                      |                      |
        |                      |<--update(1)----------|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |<--render view--------|                      |                      |                      |                      |
        |  (state preserved)   |                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |
```

## Notes

- Sessions can be reconnected to at any time before session timeout (default 24h)
- Session state (variables, presenters, watches) is preserved until session timeout
- Session timeout is configurable via `--session-timeout` (0 = never expires)
- Session URL remains valid as long as session exists; reconnection is seamless
- SharedWorker may not survive page refresh; new WebSocket is established on reconnect
- Multiple tabs can connect to same session simultaneously
- Session ID is in URL path, making URLs bookmarkable and shareable
