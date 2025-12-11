# FrontendApp

**Source Spec:** libraries.md, interfaces.md

## Responsibilities

### Knows
- sessionId: Current session ID from URL
- sharedWorker: Reference to SharedWorker instance
- rootVariable: Variable 1 reference
- isMainTab: Whether this tab is the primary connection
- appView: AppView instance for ui-app element

### Does
- initialize: Parse URL, connect to SharedWorker, create AppView for ui-app element
- handleBootstrap: Process initial viewdefs from variable 1
- handleVariableUpdate: Process incoming variable updates
- sendMessage: Send protocol message via SharedWorker
- navigateTo: Trigger SPA navigation
- handleTabActivation: Process tab activation request
- showNotification: Display desktop notification

## Collaborators

- SharedWorker: Backend communication
- SPANavigator: History management
- AppView: Renders root app via ui-app element
- ViewRenderer: View display (via AppView)
- WidgetBinder: Widget bindings

## Sequences

- seq-bootstrap.md: App initialization
- seq-frontend-connect.md: Connection establishment
- seq-activate-tab.md: Tab activation handling
