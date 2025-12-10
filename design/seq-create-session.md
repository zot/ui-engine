# Sequence: Create Session

**Source Spec:** interfaces.md
**Use Case:** Creating a new session when user accesses the site

## Participants

- Browser: User's web browser
- HTTPEndpoint: HTTP request handler
- SessionManager: Session lifecycle management
- VariableStore: Variable storage
- AppPresenter: Root app presenter

## Sequence

```
     Browser            HTTPEndpoint          SessionManager         VariableStore          AppPresenter
        |                      |                      |                      |                      |
        |---GET /------------->|                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---createSession()--->|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---generateId()------>|                      |
        |                      |                      |                      |                      |
        |                      |                      |<--sessionId----------|                      |
        |                      |                      |                      |                      |
        |                      |                      |---create(null,------>|                      |
        |                      |                      |    appData,props)    |                      |
        |                      |                      |                      |                      |
        |                      |                      |                      |---create()---------->|
        |                      |                      |                      |                      |
        |                      |                      |                      |<--presenter----------|
        |                      |                      |                      |                      |
        |                      |                      |<--variable 1---------|                      |
        |                      |                      |                      |                      |
        |                      |                      |---storeSession------>|                      |
        |                      |                      |                      |                      |
        |                      |<--sessionId----------|                      |                      |
        |                      |                      |                      |                      |
        |<--302 /sessionId-----|                      |                      |                      |
        |                      |                      |                      |                      |
        |---GET /sessionId---->|                      |                      |                      |
        |                      |                      |                      |                      |
        |<--index.html---------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Root URL redirects to session-specific URL
- Session ID embedded in URL path
- Variable 1 created as root with AppPresenter data
- Session URL can be bookmarked for reconnection
- Frontend app bootstraps after receiving HTML
