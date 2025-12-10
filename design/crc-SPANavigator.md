# SPANavigator

**Source Spec:** libraries.md, interfaces.md

## Responsibilities

### Knows
- historyIndex: Bound to AppPresenter.historyIndex
- url: Bound to AppPresenter.url
- basePath: Session URL prefix

### Does
- bindToApp: Watch historyIndex and url variables
- handleHistoryChange: Respond to historyIndex/url updates
- pushState: Add entry to browser history
- replaceState: Replace current history entry
- go: Navigate to specific history index
- handlePopState: Process browser back/forward
- buildFullUrl: Construct URL with session prefix

## Collaborators

- FrontendApp: Main app integration
- AppPresenter: State binding
- Router: URL path resolution
- ViewRenderer: Triggers view updates

## Sequences

- seq-spa-navigate.md: Navigation flow
- seq-navigate-page.md: Page change handling
- seq-navigate-url.md: URL-based navigation
