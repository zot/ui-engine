# AppPresenter

**Source Spec:** main.md

## Responsibilities

### Knows
- url: Current URL path
- historyIndex: Current position in history stack
- history: Array of page objects (presenter references)

### Does
- currentPage: Return history[historyIndex]
- navigate: Update url, push to history, increment historyIndex
- back: Decrement historyIndex if > 0
- forward: Increment historyIndex if < history.length - 1
- go: Navigate to specific history index
- replaceCurrentPage: Replace history[historyIndex] without pushing

## Collaborators

- Presenter: Base presenter functionality
- Router: Handles URL-to-presenter mapping
- SPANavigator: Frontend navigation integration
- Session: Associated with session's main app state

## Sequences

- seq-navigate-page.md: Page navigation flow
- seq-spa-navigate.md: Frontend SPA navigation
- seq-app-startup.md: App initialization
