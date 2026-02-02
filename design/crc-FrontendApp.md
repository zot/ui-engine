# FrontendApp

**Source Spec:** libraries.md, interfaces.md, js-api.md
**Requirements:** R26, R27, R28, R29, R30, R31

## Responsibilities

### Knows
- sessionId: Current session ID from URL
- sharedWorker: Reference to SharedWorker instance
- rootVariable: Variable 1 reference
- isMainTab: Whether this tab is the primary connection
- appView: AppView instance for ui-app element
- binding: BindingEngine instance for widget lookup

### Does
- initialize: Parse URL, connect to SharedWorker, create AppView for ui-app element, expose as window.uiApp (R26)
- handleBootstrap: Process initial viewdefs from variable 1
- handleVariableUpdate: Process incoming variable updates
- sendMessage: Queue protocol message via FrontendOutgoingBatcher
- navigateTo: Trigger SPA navigation
- handleTabActivation: Process tab activation request
- showNotification: Display desktop notification
- updateValue(elementId, value?): Update element's ui-value binding variable (R27-R31)

## Collaborators

- SharedWorker: Backend communication
- FrontendOutgoingBatcher: Throttled outgoing message batching
- SPANavigator: History management
- AppView: Renders root app via ui-app element
- ViewRenderer: View display (via AppView)
- WidgetBinder: Widget bindings

## Sequences

- seq-bootstrap.md: App initialization
- seq-frontend-connect.md: Connection establishment
- seq-activate-tab.md: Tab activation handling
- seq-frontend-outgoing-batch.md: Outgoing message batching
