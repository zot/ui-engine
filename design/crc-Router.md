# Router

**Source Spec:** interfaces.md

## Responsibilities

### Knows
- routes: Map of URL path pattern to presenter variable ID
- sessionId: Current session context
- basePath: Session URL prefix (/SESSION-ID)

### Does
- register: Associate URL path with presenter
- unregister: Remove URL path mapping
- resolve: Find presenter for URL path
- match: Check if URL matches registered pattern
- buildUrl: Construct full URL for presenter
- parseUrl: Extract session ID and path from URL
- isRegisteredPath: Check if path was explicitly registered by backend

## Collaborators

- SessionManager: Session-scoped routing
- AppPresenter: Navigation target
- SPANavigator: Frontend navigation
- MCPTool: Path registration via MCP

## Sequences

- seq-navigate-url.md: URL routing flow
- seq-spa-navigate.md: SPA history management
- seq-activate-tab.md: Tab activation with path
