# ViewRenderer

**Source Spec:** libraries.md

## Responsibilities

### Knows
- rootElement: DOM element for view display
- currentViewdef: Currently rendered viewdef
- activeElements: Map of variable ID to bound elements

### Does
- render: Display viewdef for current page presenter
- clear: Remove current view content
- createElements: Parse viewdef HTML and create DOM elements
- bindElements: Apply ui-* bindings to created elements
- handleViewChange: Re-render when presenter type changes
- renderViewList: Handle ui-viewlist array rendering
- renderNestedView: Handle ui-view object rendering
- updateDynamicContent: Handle ui-content HTML updates

## Collaborators

- FrontendApp: Triggers rendering
- ViewdefStore: Retrieves viewdefs
- BindingEngine: Applies bindings
- WidgetBinder: Widget-specific rendering

## Sequences

- seq-render-view.md: Full render cycle
- seq-bind-element.md: Element binding
- seq-bootstrap.md: Initial render
