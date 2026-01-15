# Widget

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- elementId: Unique ID for the bound element (vended via ElementIdVendor if element has no ID)
- variables: Map of binding name to variable ID for all bindings on this element
- unbindHandlers: Map of binding name to cleanup function (`Map<string, () => void>`)
- autoVendedId: Boolean indicating if ID was auto-assigned (for cleanup)
- scrollOnOutput: If true, scroll element to bottom when child content renders (set via path property)

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
- scrollToBottom: Scroll element to bottom if scrollable (`scrollHeight > clientHeight`)
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

## All Bindings Create Widgets

Every binding type (`ui-value`, `ui-attr-*`, `ui-view`, `ui-viewlist`, etc.) creates and registers a Widget. This is necessary because:
- Any element could become a scroll container via CSS
- Scroll-related behavior (`scrollOnOutput`) is managed at the Widget level
- Consistent cleanup and lifecycle management across all binding types

## scrollOnOutput Behavior

`scrollOnOutput` is a **universal path property** supported by all binding types (see crc-BindingEngine.md). Since any element could be a scroll container via CSS, this property applies to the widget, not the variable.

When `scrollOnOutput` is set on a widget (via path property like `?scrollOnOutput`):
1. Child content changes trigger scroll notifications:
   - Views/ViewLists notify their parent after rendering
   - `ui-value` updates on content-resizable elements (span, div, p, etc.) notify their parent
   - Input elements (input, textarea, sl-input, sl-textarea) do NOT notify (fixed dimensions)
2. Notifications bubble up through the variable hierarchy
3. When a widget with `scrollOnOutput` is found, it calls `scrollToBottom()`
4. Bubbling stops at the scrolling widget

**Examples across binding types:**
```html
<div ui-value="log?scrollOnOutput"></div>
<div ui-view="messages?scrollOnOutput"></div>
<div ui-viewlist="items?scrollOnOutput"></div>
<div ui-attr-class="theme?scrollOnOutput"></div>
```

## Collaborators

- ElementIdVendor: Global vendor for unique element IDs
- BindingEngine: Creates Widgets when processing ui-* attributes, registers unbind handlers
- Variable: Stores elementId reference to Widget (not direct DOM reference)

## Sequences

- seq-bind-element.md: Widget creation during element binding
