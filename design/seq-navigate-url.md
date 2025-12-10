# Sequence: Navigate URL

**Source Spec:** interfaces.md
**Use Case:** URL-based navigation to registered presenter

## Participants

- Browser: User's browser
- SPANavigator: SPA history manager
- Router: URL path resolver
- AppPresenter: App state holder
- ViewRenderer: View display

## Sequence

```
     Browser           SPANavigator              Router            AppPresenter          ViewRenderer
        |                      |                      |                      |                      |
        |---popstate event---->|                      |                      |                      |
        |   (back/forward)     |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---parseUrl()-------->|                      |                      |
        |                      |   (extract path)     |                      |                      |
        |                      |                      |                      |                      |
        |                      |---resolve(path)----->|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---isRegistered?----->|                      |
        |                      |                      |                      |                      |
        |                      |          [if registered path]               |                      |
        |                      |                      |<--presenterVar-------|                      |
        |                      |                      |                      |                      |
        |                      |<--presenterVar-------|                      |                      |
        |                      |                      |                      |                      |
        |                      |---navigate(var)----->|                      |                      |
        |                      |                      |---setHistoryIndex--->|                      |
        |                      |                      |   (or push)          |                      |
        |                      |                      |                      |                      |
        |                      |                      |<--currentPage()------|                      |
        |                      |                      |                      |                      |
        |                      |                      |---render()---------->|                      |
        |                      |                      |                      |---display view------>|
        |                      |                      |                      |                      |
        |          [if not registered]                |                      |                      |
        |                      |<--null---------------|                      |                      |
        |                      |                      |                      |                      |
        |<--404 or ignore------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

- Only backend-registered paths are navigable via URL
- Unregistered paths show error or are ignored
- Navigation updates AppPresenter history state
- URL changes without full page reload (SPA style)
- Back/forward triggers popstate event handling
