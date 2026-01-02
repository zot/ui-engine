# View

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- element: Container DOM element for this view
- htmlId: Unique frontend-vended HTML id
- variable: Variable bound to this view (object reference)
- rendered: Whether view has been successfully rendered

### Does
- create: Initialize view from element with ui-view attribute, set variable namespace properties
- render: Render variable using 3-tier namespace resolution, returns boolean
- setVariable: Update bound variable (triggers re-render)
- clear: Remove rendered content from element
- getHtmlId: Return unique HTML id
- markPending: Add to pending views list (missing type or viewdef)
- removePending: Remove from pending views list after successful render
- resolveNamespace: Apply 3-tier resolution (namespace -> fallbackNamespace -> DEFAULT)

## Collaborators

- ViewdefStore: Retrieves viewdefs by TYPE.NAMESPACE
- ViewRenderer: Creates and manages Views
- BindingEngine: Applies bindings to rendered content
- Variable: Source of object reference and type property

## Notes

### Default Access Property

The `ui-view` binding automatically adds `access=r` (read-only) if no `access` property is specified. Views are typically read-only bindings that display object references.

### Namespace Property Setting

When creating a view's variable:
1. If `ui-namespace` attribute is specified, set the variable's `namespace` property to that value
2. If no `ui-namespace` attribute, inherit `namespace` property from the parent variable (if set)
3. Always inherit `fallbackNamespace` property from the parent variable (if set)

### 3-Tier Namespace Resolution

When rendering, viewdef lookup uses this resolution order:
1. If variable has `namespace` property and `TYPE.{namespace}` viewdef exists, use it
2. Otherwise, if variable has `fallbackNamespace` property and `TYPE.{fallbackNamespace}` viewdef exists, use it
3. Otherwise, use `TYPE.DEFAULT`

This allows custom namespaces to fall back gracefully when specific viewdefs don't exist.

## Sequences

- seq-render-view.md: View creation and rendering with namespace resolution
- seq-viewdef-delivery.md: Processing pending views when viewdefs arrive
