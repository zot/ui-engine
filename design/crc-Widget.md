# Widget

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- elementId: Unique ID for the bound element (vended via ElementIdVendor if element has no ID)
- variables: Map of binding name to variable ID for all bindings on this element
- unbindHandlers: Map of binding name to cleanup function (`Map<string, () => void>`)
- autoVendedId: Boolean indicating if ID was auto-assigned (for cleanup)
- viewElementId: (optional) Element ID of the containing view for hot-reload support

### Does
- create: Create a Widget for an element with ui-* bindings
- vendElementId: Request ID from ElementIdVendor if element lacks one
- registerBinding: Add a binding name to variable ID mapping
- unregisterBinding: Remove a binding from the variables map
- addUnbindHandler: Register a cleanup function for a binding (`addUnbindHandler(name, fn)`)
- unbindAll: Call all unbind handlers and clear the unbindHandlers map
- getVariableId: Get variable ID for a binding name
- hasBinding: Check if a binding exists by name (e.g., "ui-value")
- getElement: Look up DOM element by elementId (via document.getElementById)
- destroy: Call unbindAll(), clean up, and remove element ID if auto-vended

## Element ID Vending

When a Widget is created for an element without an ID:
1. Requests an ID from global ElementIdVendor (`vendId()`)
2. Assigns the vended ID as the element's ID
3. Sets `autoVendedId = true` (for cleanup)

Format: `ui-{counter}` (e.g., `ui-1`, `ui-2`) - provided by global ElementIdVendor

## Variable-Widget Relationship

Variables created by bindings store a reference to their Widget via the element ID:
- Variables do NOT store direct DOM element references
- Element lookup uses `document.getElementById(elementId)` when needed
- This enables proper cleanup and avoids memory leaks from circular references
- Allows serialization of binding state if needed

## Why Element ID (Not Direct Reference)

Storing element ID instead of direct DOM reference:
- **Avoids circular references**: Element -> Widget -> Variables -> Element would create cycles
- **Prevents memory leaks**: DOM elements can be garbage collected when removed
- **Enables serialization**: Binding state can be serialized for debugging/persistence
- **Decouples binding from DOM**: Variables work with IDs, DOM lookup happens on demand

## Binding Ownership

Widget is the sole owner of all bindings for an element. There is no separate Binding interface - the Widget directly manages cleanup via `unbindHandlers`:

1. When a binding is created, BindingEngine registers its cleanup function via `addUnbindHandler(name, cleanupFn)`
2. The cleanup function handles: removing event listeners, unwatching variables, destroying child variables
3. When unbinding the element, BindingEngine calls `widget.unbindAll()` which invokes all cleanup functions
4. Each binding name has exactly one cleanup handler (e.g., "ui-value", "ui-attr-hidden", "ui-event-click")

## Collaborators

- ElementIdVendor: Global vendor for unique element IDs
- BindingEngine: Creates Widgets when processing ui-* attributes, registers unbind handlers
- Variable: Stores elementId reference to Widget (not direct DOM reference)
- View: Widgets track their containing view for hot-reload targeting

## Sequences

- seq-bind-element.md: Widget creation during element binding
