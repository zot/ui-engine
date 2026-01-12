# View

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- elementId: ID of container element for this view (NOT direct DOM reference)
- variable: Variable bound to this view (object reference)
- rendered: Whether view has been successfully rendered
- viewdefKey: The resolved viewdef key (e.g., "Contact.COMPACT") stored as `data-ui-viewdef` attribute

### Does
- create: Initialize view from element with ui-view attribute, vend element ID if needed, set variable namespace properties, register widget
- render: Render variable using 3-tier namespace resolution, set `data-ui-viewdef` attribute, returns boolean
- setVariable: Update bound variable (triggers re-render)
- clear: Remove rendered content from element (unbinds existing widgets)
- getElement: Look up DOM element by elementId (via document.getElementById)
- markPending: Add to pending views list (missing type or viewdef)
- removePending: Remove from pending views list after successful render
- resolveNamespace: Apply 3-tier resolution (namespace -> fallbackNamespace -> DEFAULT)
- rerender: Hot-reload re-render using updated viewdef (unbinds old widgets, re-binds new)
- notifyParentRendered: After rendering, add parent variable ID to BindingEngine's pendingScrollNotifications set

## Collaborators

- ElementIdVendor: Vends unique element ID if element lacks one
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

## Notes

### Hot-Reload Support

Views support hot-reload re-rendering:
1. `data-ui-viewdef` attribute on container element stores the resolved viewdef key
2. When updated viewdefs arrive, ViewdefStore queries `[data-ui-viewdef="KEY"]` to find views
3. Each matching view's `rerender()` method is called with the updated viewdef
4. Re-rendering reuses the same variable and container element
5. Widgets within the view are unbound via `clear()` and recreated during re-render

## Widget Registration

Views register themselves with the BindingEngine's widgets map. This enables:
- Consistent cleanup and lifecycle management
- `scrollOnOutput` support (set on the widget, not the view)
- Hot-reload targeting

When `scrollOnOutput` is specified in the path (e.g., `ui-view="chatLog?scrollOnOutput"`), it is set on the element's widget, not on the View itself. See crc-Widget.md for details.

## Render Notifications

After a view renders, it notifies its parent so ancestor widgets with `scrollOnOutput` can scroll:

1. After rendering, call `notifyParentRendered()` which adds the parent variable ID to BindingEngine's `pendingScrollNotifications` set
2. The BindingEngine processes these notifications after the batch completes (see crc-BindingEngine.md)
3. If an ancestor widget has `scrollOnOutput`, it scrolls to bottom

This batched approach ensures multiple child renders cause only one scroll.

## Sequences

- seq-render-view.md: View creation and rendering with namespace resolution
- seq-viewdef-delivery.md: Processing pending views when viewdefs arrive
- seq-viewdef-hotload.md: Hot-reload re-rendering when viewdefs change
