# AppView

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- elementId: ID of element with ui-app attribute (NOT direct DOM reference)
- variableId: Always 1 (root app variable)
- view: View instance that renders the app content
- namespace: Viewdef namespace (default: DEFAULT)

### Does
- initialize: Find ui-app element, vend element ID if needed, create View, watch variable 1
- render: Delegate to View when variable 1 updates with type property
- getElement: Look up DOM element by elementId (via document.getElementById)
- destroy: Cleanup View and watchers

## Collaborators

- ElementIdVendor: Vends unique element ID if ui-app element lacks one
- View: Renders variable 1 using TYPE.NAMESPACE viewdef
- ViewdefStore: Retrieves viewdefs, manages pending views
- VariableStore: Provides variable 1 data and updates
- ViewRenderer: Provides binding callback for rendered content

## Sequences

- seq-bootstrap.md: App initialization including AppView setup
- seq-render-view.md: View rendering cycle (delegated)
