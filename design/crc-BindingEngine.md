# BindingEngine

**Source Spec:** viewdefs.md, libraries.md

## Responsibilities

### Knows
- widgets: Map of element ID to Widget for bound elements (sole tracking structure for bindings)
- store: VariableStore for creating/watching child variables
- inputElements: Set of element types that support two-way value binding (`input`, `textarea`, `sl-input`, `sl-textarea`)
- pendingScrollNotifications: Set of variable IDs whose widgets should be checked for scrollOnOutput

### Does
- bind: Apply all ui-* bindings to an element (creates Widget if needed)
- unbindElement: Call widget.unbindAll() and remove Widget from widgets map
- getOrCreateWidget: Get existing Widget for element or create new one (vends element ID if needed)
- createValueBinding: Create ui-value binding with child variable, register unbind handler with Widget
- createKeypressBinding: Create ui-keypress binding (shorthand for ui-value with keypress option)
- createAttrBinding: Create ui-attr-* binding with child variable, register unbind handler with Widget
- createClassBinding: Create ui-class-* binding with child variable, register unbind handler with Widget
- createStyleBinding: Create ui-style-* binding with child variable, register unbind handler with Widget
- createCodeBinding: Create ui-code binding with child variable, register unbind handler with Widget
- createEventBinding: Create ui-event-* binding, register unbind handler with Widget (passes widget reference)
- createKeypressEventBinding: Create ui-event-keypress-* binding, register unbind handler with Widget
- createActionBinding: Create ui-action binding, register unbind handler with Widget
- sendVariableUpdate: Send variable update with local value setting and duplicate suppression (checks if update should be skipped, sets local value first, then sends)
- shouldSuppressUpdate: Check if update should be skipped due to duplicate value (see Duplicate Update Suppression)
- parsePath: Parse path with optional URL-style properties (?prop=value); properties without values default to `true`; separates universal properties (handled locally) from variable properties (sent to backend)
- parseKeypressAttribute: Extract target key and modifiers from ui-event-keypress-* attribute name (returns `{key, modifiers}`)
- normalizeKeyName: Convert attribute key name to browser event.key value (e.g., "enter" -> "Enter")
- isModifierKey: Check if a segment is a known modifier (ctrl, shift, alt, meta)
- selectInputEvent: Choose event type for input elements (`blur` by default, `input` if `keypress` property is set or `ui-keypress` attribute used)
- integrateWidgetBinding: Coordinate with WidgetBinder for widget-specific value handling
- determineDefaultAccess: Determine default `access` property based on binding type and element
- addScrollNotification: Add a variable ID to pendingScrollNotifications set (called by Views after rendering)
- processScrollNotifications: Process pending notifications after batch completes using current/next pattern

## Universal Path Properties

All bindings support these path properties regardless of binding type:

- `scrollOnOutput` - Sets `widget.scrollOnOutput = true`. Handled locally by BindingEngine, not sent to backend. Any element can be a scroll container via CSS.
- `access` - Override default access mode (`r`, `rw`, `w`, `action`). Sent to backend as variable property.

**Binding-specific properties** (only meaningful for certain bindings):
- `keypress` - For ui-value on input elements
- `wrapper`, `item`, `create` - For ui-view/ui-viewlist

**Processing flow:**
1. `parsePath()` extracts all properties from `?key=value` syntax
2. Universal properties like `scrollOnOutput` are applied to the widget
3. Remaining properties are set on the child variable for backend processing

## Child Variable Architecture (Server-Side Path Resolution)

**Critical: All path-based bindings MUST create child variables for backend path resolution.**

Variable values sent to the frontend are **object references** (e.g., `{"obj": 1}`), not actual object contents. This means:
- Client-side path resolution is **impossible** - the frontend cannot extract `isActive` from `{"obj": 1}`
- All paths must be resolved by the backend, which has access to actual object data
- Every binding creates a **child variable** with a `path` property that the backend resolves

**Implementation pattern for ALL binding types (ui-value, ui-attr-*, ui-class-*, ui-style-*, ui-code):**

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

## Default Access Property

Bindings automatically set `access=r` (read-only) unless explicitly overridden:

| Binding Type | Default Access | Notes |
|--------------|----------------|-------|
| `ui-value` on interactive elements | `rw` | input, textarea, select, sl-input, sl-textarea, sl-select |
| `ui-value` on non-interactive elements | `r` | div, span, etc. |
| `ui-attr-*` | `r` | Attribute bindings are read-only |
| `ui-class-*` | `r` | Class bindings are read-only |
| `ui-style-*` | `r` | Style bindings are read-only |
| `ui-code` | `r` | Code execution bindings are read-only |
| `ui-view` | `r` | View bindings are read-only |
| `ui-viewlist` | `r` | ViewList bindings are read-only |

The `determineDefaultAccess` method checks the binding type and element tag to determine the appropriate default.

## ui-code Binding

The `ui-code` binding executes JavaScript code when the bound variable's value changes.

**Attribute format:**
```html
<div ui-code="codePath">...</div>
```

**Behavior:**
1. Creates a child variable with `path` property set to the attribute value
2. When the child variable receives an update, the value is treated as JavaScript code
3. The code is executed with four variables in scope:
   - `element` - the bound DOM element (looked up by element ID, not a stored reference)
   - `value` - the new value from the variable
   - `variable` - the variable for this binding (provides access to widget via properties)
   - `store` - the VariableStore for accessing/creating other variables

**Example:**
```html
<div ui-code="highlightCode"></div>
```

When the `highlightCode` variable changes to `"element.classList.add('highlight')"`, that code executes with `element` being the div.

**Advanced example (using variable and store):**
```javascript
// Code can access the widget via variable properties
const widgetId = variable.properties.elementId;
// Code can create or access other variables
const otherVar = store.get(someVarId);
```

**Security note:** The code is executed using `new Function()` with controlled scope. Only use with trusted backend code.

## Keypress Event Binding

The `ui-event-keypress-*` attribute creates an event binding that fires only when a specific key (with optional modifiers) is pressed.

**Attribute format:**
```
ui-event-keypress-{modifiers}-{key}
```

**Examples:**
```html
<input ui-event-keypress-enter="onEnterPressed">
<div ui-event-keypress-escape="onEscape" tabindex="0">
<input ui-event-keypress-ctrl-enter="submitForm">
<input ui-event-keypress-ctrl-shift-s="saveAll">
<div ui-event-keypress-alt-left="navigateBack" tabindex="0">
```

**Processing:**
1. `parseKeypressAttribute(attrName)` extracts the key name and modifiers from the attribute:
   - Splits the attribute suffix on `-` (e.g., `ctrl-shift-s` -> `["ctrl", "shift", "s"]`)
   - Uses `isModifierKey()` to identify modifiers (ctrl, shift, alt, meta)
   - The last non-modifier segment is the key
   - Returns `{key: "s", modifiers: Set{"ctrl", "shift"}}`
2. `normalizeKeyName(keyName)` converts it to the browser's `event.key` format:
   - `enter` -> `Enter`
   - `escape` -> `Escape`
   - `left` -> `ArrowLeft`
   - `right` -> `ArrowRight`
   - `up` -> `ArrowUp`
   - `down` -> `ArrowDown`
   - `tab` -> `Tab`
   - `space` -> ` ` (space character)
   - Single letters remain as-is (case-insensitive matching)
3. `createKeypressEventBinding(element, path, targetKey, modifiers)` creates an EventBinding that:
   - Listens to `keydown` events on the element
   - Filters by the normalized target key
   - Filters by exact modifier match (all specified modifiers pressed, no extra modifiers)
   - Updates the variable with the key name when matched

**Modifier matching:**
The `matchesModifiers(event, requiredModifiers)` function checks:
- `event.ctrlKey` matches `"ctrl" in requiredModifiers`
- `event.shiftKey` matches `"shift" in requiredModifiers`
- `event.altKey` matches `"alt" in requiredModifiers`
- `event.metaKey` matches `"meta" in requiredModifiers`

All four conditions must be true for an exact match. This ensures `ctrl-s` does not match Ctrl+Shift+S.

**Example:**
```html
<input ui-event-keypress-ctrl-enter="submitForm">
```
When the user presses Ctrl+Enter (without Shift, Alt, or Meta) in the input, the `submitForm` variable is set to `"enter"`.

## Local Value Setting (Universal Principle)

**Spec: viewdefs.md "Frontend Update Behavior"**

**Whenever the frontend sends a variable update to the backend, it MUST first set the value in the local variable cache.** This ensures the frontend's cached variable state accurately reflects the UI state being sent to the backend.

**Pattern:**
```typescript
// sendVariableUpdate(variable, newValue, pathOptions)
if (shouldSuppressUpdate(variable, newValue, pathOptions)) {
  return;  // Skip duplicate updates
}
variable.value = newValue;  // MUST set local value first
protocolHandler.send({type: 'update', id: variable.id, value: newValue});
```

**Benefits:**
- Immediate UI feedback (no waiting for backend round-trip)
- Consistency between local state and pending backend state
- Enables optimistic updates with eventual consistency
- Frontend cache always reflects what was sent to backend

**Used by:**
- ValueBinding: When element value changes
- EventBinding: When syncing value before event (via syncValueBinding)
- Any code sending variable updates to backend

## Duplicate Update Suppression

**Spec: viewdefs.md "Frontend Update Behavior"**

Bindings that do NOT have `access=action` or `access=w` MUST NOT send an update if the variable's value has not changed.

**Pattern:**
```typescript
// shouldSuppressUpdate(variable, newValue, pathOptions): boolean
if (pathOptions.access === 'action' || pathOptions.access === 'w') {
  return false;  // Never suppress actions or write-only
}
return variable.value === newValue;  // Suppress if unchanged
```

**When to suppress:**
- Before sending an update, compare the new value to the variable's current cached value
- If values are equal, skip the update entirely (do not set local value, do not send message)

**When NOT to suppress (always send):**
- `access=action` - Actions are side-effect triggers, always send
- `access=w` - Write-only bindings, always send

**Rationale:**
- Reduces unnecessary network traffic
- Prevents redundant backend processing
- Blur events may fire without actual value changes
- Action bindings intentionally trigger side effects regardless of value

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

## Widget-Based Binding Ownership

Widget is the sole owner of all bindings for an element. There is no separate Binding interface - cleanup is managed via `widget.unbindHandlers`:

**Creating a binding:**
1. BindingEngine creates child variable and sets up watch/listener
2. BindingEngine creates a cleanup function that: removes listeners, unwatches variable, destroys child variable
3. BindingEngine registers the cleanup function: `widget.addUnbindHandler(bindingName, cleanupFn)`

**Unbinding an element:**
1. BindingEngine calls `widget.unbindAll()` - invokes all cleanup handlers
2. BindingEngine removes Widget from `widgets` map
3. Widget removes auto-vended element ID if applicable

**Benefits:**
- No separate Binding interface to maintain
- Widget encapsulates all cleanup for an element
- Single point of control for binding lifecycle

## Scroll Notification Processing

When Views/ViewLists render, they notify the BindingEngine so ancestor widgets with `scrollOnOutput` can scroll. This is batched to avoid multiple scrolls during a single update cycle.

**Adding notifications:**
- Views/ViewLists call `addScrollNotification(parentVarId)` after rendering
- The variable ID is added to `pendingScrollNotifications` set

**Processing notifications (after batch completes):**

```
processScrollNotifications():
  current = pendingScrollNotifications
  next = new Set()

  while current is not empty:
    for each varId in current:
      widget = widgets.get(store.get(varId)?.properties.elementId)
      if widget?.scrollOnOutput:
        widget.scrollToBottom()
        // Don't bubble further
      else:
        parentId = store.get(varId)?.parentId
        if parentId:
          next.add(parentId)

    current.clear()
    swap current and next

  pendingScrollNotifications.clear()
```

**Key behaviors:**
- Multiple child renders in one batch cause only one scroll
- Scrolling happens at the correct ancestor widget (the one with `scrollOnOutput`)
- Views inside ViewLists trigger scrolling on the ViewList's widget or any ancestor
- Processing stops when a widget scrolls (doesn't bubble further up)
- Any binding type can have `scrollOnOutput` since any element could be a scroll container via CSS

## Collaborators

- ElementIdVendor: Global vendor for unique element IDs
- Widget: Binding context for elements with ui-* bindings (element ID, variable map, unbind handlers)
- Viewdef: Source of binding directives
- Variable: Target of bindings
- VariableStore: Provides watch() for subscribing to variable changes
- View: Handles ui-view bindings
- ViewList: Handles ui-viewlist bindings
- WidgetBinder: Widget-specific value binding (called by BindingEngine for Shoelace elements)

## Sequences

- seq-bind-element.md: Element binding process
- seq-handle-event.md: Event to variable flow
- seq-handle-keypress-event.md: Keypress event binding flow
- seq-render-view.md: Full view rendering
- seq-viewlist-update.md: ViewList array updates
- seq-input-value-binding.md: Input element two-way binding with event selection
