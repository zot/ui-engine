# Sequence: Frontend Reconnect

**Source Spec:** interfaces.md
**Use Case:** Browser reconnecting to session within grace period after disconnect (e.g., page refresh)

## Participants

- Browser: User's browser tab
- FrontendApp: Frontend application
- SharedWorker: Tab coordination worker
- WebSocketEndpoint: Server WebSocket handler
- Session: Session instance
- SessionManager: Session management

## Sequence

### Disconnect Flow (Grace Period Start)

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
        |                      |                      |                      |          [if last frontend]
        |                      |                      |                      |                      |
        |                      |                      |                      |<-startConnectionTimeout
        |                      |                      |                      |                      |
        |                      |                      |                      |     [start 5s timer] |
        |                      |                      |                      |                      |
```

### Reconnect Flow (Within Grace Period)

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
        |                      |                      |                      |---sessionExists?---->|                      |
        |                      |                      |                      |                      |---getSession-------->|
        |                      |                      |                      |                      |<--session (in grace)-|
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |---isInGracePeriod?-->|                      |
        |                      |                      |                      |<--true---------------|                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |---onReconnect------->|                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |            cancelConnectionTimeout          |
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

### Timeout Expiration Flow

```
                                                      WebSocketEndpoint          Session          SessionManager
                                                             |                      |                      |
                                                             |     [5s timer fires] |                      |
                                                             |                      |                      |
                                                             |<--onTimeout----------|                      |
                                                             |                      |                      |
                                                             |                      |---onConnectionTimeout>|
                                                             |                      |                      |
                                                             |                      |          [may destroy session]
                                                             |                      |          [depends on session timeout]
                                                             |                      |                      |
```

## Notes

- Grace period default is 5 seconds (configurable via `--connection-timeout`)
- Session state (variables, presenters, watches) is preserved during grace period
- Only frontend disconnects trigger grace period; backend disconnects do not
- If multiple frontends are connected, grace period only starts when last frontend disconnects
- Session URL remains valid during grace period; reconnection is seamless
- After grace period expires, session may be cleaned up (depends on session timeout setting)
- If session timeout is set to 0 (never), session persists indefinitely after grace period
- SharedWorker may not survive page refresh; new WebSocket is established on reconnect
