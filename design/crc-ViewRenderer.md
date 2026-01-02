# ViewRenderer

**Source Spec:** viewdefs.md, libraries.md

## Responsibilities

### Knows
- rootElement: DOM element for view display
- currentViewdef: Currently rendered viewdef
- activeElements: Map of variable ID to bound elements
- nextHtmlId: Counter for vending unique HTML ids

### Does
- render: Render variable with namespace, returns boolean success
- clear: Remove current view content
- createElements: Parse viewdef HTML and create DOM elements
- bindElements: Apply ui-* bindings to created elements
- handleViewChange: Re-render when presenter type changes
- createView: Create View for ui-view element
- createViewList: Create ViewList for ui-viewlist element
- updateDynamicContent: Handle ui-content HTML updates
- vendHtmlId: Generate unique HTML id for views
- lookupViewdef: Get viewdef using 3-tier resolution (namespace -> fallbackNamespace -> DEFAULT)

## Collaborators

- FrontendApp: Triggers rendering
- AppView: Uses ViewRenderer for root app rendering
- ViewdefStore: Retrieves viewdefs, manages pending views
- BindingEngine: Applies bindings
- WidgetBinder: Widget-specific rendering
- View: Individual object reference views
- ViewList: Array of object reference views

## Sequences

- seq-render-view.md: Full render cycle with View/ViewList
- seq-viewlist-update.md: ViewList array updates
- seq-bind-element.md: Element binding
- seq-bootstrap.md: Initial render
