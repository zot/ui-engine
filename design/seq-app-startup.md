# Sequence: App Startup

**Source Spec:** main.md, interfaces.md
**Use Case:** Complete application startup flow

## Participants

- User: End user
- Browser: User's browser
- UIServer: UI server process
- SessionManager: Session management
- AppPresenter: Root app presenter
- Frontend: Browser frontend app

## Sequence

```
     User               Browser              UIServer          SessionManager          AppPresenter            Frontend
        |                      |                      |                      |                      |                      |
        |---open URL---------->|                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---GET /------------->|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |---createSession()--->|                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |---create()---------->|                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |---init(history,----->|
        |                      |                      |                      |                      |    url,historyIdx)   |
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |<--appPresenter-------|                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |<--sessionId----------|                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |<--302 /sessionId-----|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---GET /sessionId---->|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |<--index.html---------|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---load JS/CSS------->|                      |                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |---init FrontendApp-->|                      |                      |                      |
        |                      |                      |                      |                      |---initialize-------->|
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |---connect()-->
        |                      |                      |                      |                      |                      |
        |                      |                      |<--WebSocket connect--|                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |---watch(1)---------->|                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |<--update(1,viewdefs)-|                      |                      |
        |                      |                      |                      |                      |                      |
        |                      |                      |----------------------------------------------update(1)------------->|
        |                      |                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |---render()-->
        |                      |                      |                      |                      |                      |
        |<--display UI---------|                      |                      |                      |                      |
        |                      |                      |                      |                      |                      |
```

## Notes

- Root URL redirects to session-specific URL
- Session creates AppPresenter with initial state
- Frontend connects and watches variable 1
- Variable 1 contains app state and viewdefs
- Initial page rendered from currentPage()
- User sees UI and can begin interaction
