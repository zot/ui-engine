# View

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- elementId: ID of first element for this view (NOT direct DOM reference)
- lastElementId: ID of last element for multi-element templates (same as elementId for single-element)
- variable: Variable bound to this view (object reference)
- rendered: Whether view has been successfully rendered
- viewdefKey: The resolved viewdef key (e.g., "Contact.COMPACT") stored as `ui-viewdef` attribute on first element

### Does
- create: Initialize view from element with ui-view attribute, vend element ID if needed, set variable namespace properties, register widget
- render: Render variable by replacing view's element(s) with template content, supports multi-element templates, returns boolean
- setVariable: Update bound variable (triggers re-render)
- clear: Remove all view elements from DOM (from elementId to lastElementId), destroy child views
- destroy: Cleanup view - unwatch, remove from pending, clear DOM, destroy associated variable
- getElement: Look up first DOM element by elementId (via document.getElementById)
- getElements: Look up all DOM elements owned by this view (from elementId to lastElementId siblings)
- markPending: Add to pending views list (missing type or viewdef)
- removePending: Remove from pending views list after successful render
- resolveNamespace: Apply 3-tier resolution (namespace -> fallbackNamespace -> DEFAULT)
- rerender: Hot-reload re-render using updated viewdef (destroys old widgets/children, re-renders)
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

### Element Replacement Model

Views replace their element(s) in the DOM rather than adding children to a container:
1. Template content replaces the view's current element(s)
2. First new element gets the stable `elementId`, additional elements get vended IDs
3. `lastElementId` tracks the last element for multi-element templates
4. On re-render: destroy old children first, remove old elements, insert new elements at same position

### Hot-Reload Support

Views support hot-reload re-rendering:
1. `ui-viewdef` attribute on first element stores the resolved viewdef key
2. When updated viewdefs arrive, ViewdefStore queries `[ui-viewdef="KEY"]` to find views
3. Each matching view's `forceRender()` method is called
4. Re-rendering reuses the same variable; old elements are replaced with new ones
5. Old child views/viewlists are destroyed before new elements are inserted (widgets keyed by elementId)

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

## Variable Destruction

When a View is destroyed, it must destroy its associated variable:
1. The variable was created when the View was set up (via `setupChildView`)
2. `destroy()` calls `VariableStore.destroy(varId)` to notify the backend
3. Backend destruction is recursive - destroys all child variables
4. This prevents variable leaks during hot-reload re-render cycles

## Sequences

- seq-render-view.md: View creation and rendering with namespace resolution
- seq-viewdef-delivery.md: Processing pending views when viewdefs arrive
- seq-viewdef-hotload.md: Hot-reload re-rendering when viewdefs change
