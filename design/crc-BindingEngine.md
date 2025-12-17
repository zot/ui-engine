# BindingEngine

**Source Spec:** viewdefs.md, libraries.md

## Responsibilities

### Knows
- activeBindings: Map of element to list of active bindings
- store: VariableStore for creating/watching child variables
- inputElements: Set of element types that support two-way value binding (`input`, `textarea`, `sl-input`, `sl-textarea`)

### Does
- bind: Apply all ui-* bindings to an element
- unbind: Remove all bindings from an element and destroy child variables
- createValueBinding: Create ui-value binding with child variable
- createAttrBinding: Create ui-attr-* binding with child variable
- createClassBinding: Create ui-class-* binding with child variable
- createStyleBinding: Create ui-style-*-* binding with child variable
- createEventBinding: Create ui-event-* binding with action variable
- createActionBinding: Create ui-action binding with action variable
- parsePath: Parse path with optional URL-style properties (?create=Type&prop=value); properties without values default to `true`
- selectInputEvent: Choose event type for input elements (`blur` by default, `input` if `keypress` property is set)
- integrateWidgetBinding: Coordinate with WidgetBinder for widget-specific value handling

## Child Variable Architecture (Server-Side Path Resolution)

**Critical: All path-based bindings MUST create child variables for backend path resolution.**

Variable values sent to the frontend are **object references** (e.g., `{"obj": 1}`), not actual object contents. This means:
- Client-side path resolution is **impossible** - the frontend cannot extract `isActive` from `{"obj": 1}`
- All paths must be resolved by the backend, which has access to actual object data
- Every binding creates a **child variable** with a `path` property that the backend resolves

**Implementation pattern for ALL binding types (ui-value, ui-attr-*, ui-class-*, ui-style-*-*):**

1. Parse the path from the attribute value
2. Create a **child variable** under the context variable with `path` property set
3. Watch the **child variable** (not the parent) for value updates
4. The backend resolves the path and sends the actual value (boolean, string, number, etc.)
5. Destroy the child variable when the binding is unbound

**Example:**
```html
<div ui-attr-hidden="isEditView">
```

The binding engine:
1. Creates child variable: `{parentId: contextVarId, properties: {path: "isEditView"}}`
2. Watches the child variable for updates
3. Backend resolves `isEditView` on the parent object and sends `true` or `false`
4. Binding updates the `hidden` attribute based on the boolean value

## Nullish Path Handling

Bindings gracefully handle nullish paths (via PathNavigator):
- **Read direction:** Display empty/default value when path segment is null/undefined (no error)
- **Write direction:** Issue `error` message with code `path-failure` when intermediate path segment is nullish, allowing UI to show error state (e.g., red border). Error clears on successful update.

Example: `ui-value="selectedContact.firstName"` works when `selectedContact` is null (shows empty).
When user attempts to edit a field with a nullish path, the field shows an error indicator until the path becomes valid.

## Input Update Behavior

By default, input elements send updates on `blur` (when the user tabs out or clicks away). This reduces network traffic and allows users to make multiple edits before committing.

To send updates on every keypress, add the `keypress` property to the path:

```html
<input ui-value="name?keypress">
<sl-input ui-value="search?keypress"></sl-input>
```

**Supported elements:** `<input>`, `<textarea>`, `<sl-input>`, `<sl-textarea>`

**Event selection:**
- Default: Listen to `blur` (native) or `sl-change` (Shoelace)
- With `keypress` property: Listen to `input` (native) or `sl-input` (Shoelace)

**Widget integration:** BindingEngine calls WidgetBinder's `bindWidget()` for Shoelace elements, passing the parsed path options including `keypress`. WidgetBinder uses these options to select the appropriate event type.

## Collaborators

- ValueBinding: Handles variable-to-element bindings
- EventBinding: Handles element-to-variable bindings
- Viewdef: Source of binding directives
- Variable: Target of bindings
- WatchManager: Subscribes to variable changes
- View: Handles ui-view bindings
- ViewList: Handles ui-viewlist bindings
- WidgetBinder: Widget-specific value binding (called by BindingEngine for Shoelace elements)

## Sequences

- seq-bind-element.md: Element binding process
- seq-handle-event.md: Event to variable flow
- seq-render-view.md: Full view rendering
- seq-viewlist-update.md: ViewList array updates
- seq-input-value-binding.md: Input element two-way binding with event selection
