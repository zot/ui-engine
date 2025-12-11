# View

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- element: Container DOM element for this view
- htmlId: Unique frontend-vended HTML id
- variable: Variable bound to this view (object reference)
- namespace: Viewdef namespace (default: DEFAULT)
- rendered: Whether view has been successfully rendered

### Does
- create: Initialize view from element with ui-view attribute
- render: Render variable using TYPE.NAMESPACE viewdef, returns boolean
- setVariable: Update bound variable (triggers re-render)
- clear: Remove rendered content from element
- getHtmlId: Return unique HTML id
- markPending: Add to pending views list (missing type or viewdef)
- removePending: Remove from pending views list after successful render

## Collaborators

- ViewdefStore: Retrieves viewdefs by TYPE.NAMESPACE
- ViewRenderer: Creates and manages Views
- BindingEngine: Applies bindings to rendered content
- Variable: Source of object reference and type property

## Sequences

- seq-render-view.md: View creation and rendering
- seq-viewdef-delivery.md: Processing pending views when viewdefs arrive
