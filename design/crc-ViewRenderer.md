# ViewRenderer

**Source Spec:** viewdefs.md, libraries.md

## Responsibilities

### Knows
- rootElementId: ID of root element for view display (NOT direct DOM reference)
- currentViewdef: Currently rendered viewdef
- activeElementIds: Map of variable ID to element IDs (NOT direct element references)

### Does
- render: Render variable with namespace, returns boolean success
- clear: Remove current view content
- createElements: Parse viewdef HTML and create DOM elements
- collectScripts: Collect script elements from cloned content before binding
- appendToElement: Append cloned nodes to container element (nodes now in DOM)
- bindElements: Apply ui-* bindings to created elements
- activateScripts: Activate collected scripts after binding (scripts are DOM-connected)
- handleViewChange: Re-render when presenter type changes
- createView: Create View for ui-view element
- createViewList: Create ViewList for ui-viewlist element
- updateDynamicContent: Handle ui-content HTML updates
- getRootElement: Look up root element by rootElementId (via document.getElementById)
- lookupViewdef: Get viewdef using 3-tier resolution (namespace -> fallbackNamespace -> DEFAULT)

## Collaborators

- ElementIdVendor: Vends unique element IDs for views and managed elements
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
