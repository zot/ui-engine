# Widget

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- elementId: Unique ID for the bound element (vended via ElementIdVendor if element has no ID)
- variables: Map of binding name to variable ID for all bindings on this element
- autoVendedId: Boolean indicating if ID was auto-assigned (for cleanup)

### Does
- create: Create a Widget for an element with ui-* bindings
- vendElementId: Request ID from ElementIdVendor if element lacks one
- registerBinding: Add a binding name to variable ID mapping
- unregisterBinding: Remove a binding from the variables map
- getVariableId: Get variable ID for a binding name
- getElement: Look up DOM element by elementId (via document.getElementById)
- destroy: Clean up all bindings and remove element ID if auto-vended

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

## Collaborators

- ElementIdVendor: Global vendor for unique element IDs
- BindingEngine: Creates Widgets when processing ui-* attributes
- Variable: Stores elementId reference to Widget (not direct DOM reference)
- ValueBinding: Uses Widget to access element and variable mappings
- EventBinding: Uses Widget to access element for event listeners

## Sequences

- seq-bind-element.md: Widget creation during element binding
