# ValueBinding

**Source Spec:** viewdefs.md, libraries.md

Value bindings connect variables to element properties. They are created by BindingEngine and registered with the Widget via unbind handlers (no separate Binding interface).

## Responsibilities

### Knows
- widget: The Widget this binding belongs to (provides element ID and variable mappings)
- elementId: ID of the bound element (from Widget, for element lookup)
- childVarId: ID of the child variable created for this binding (NOT the parent context variable)
- bindingType: One of value, keypress, attr, class, style, code
- attributeName: For attr/class/style, the specific attribute
- path: Path property value sent to backend for resolution
- pathOptions: Parsed path options including `keypress`, `create`, `wrapper`, `access`, `scrollOnOutput`, etc.
- defaultValue: Empty/default value for nullish paths (empty string, false, etc.)
- updateEvent: Event to listen for updates (`blur`/`input` for native, `sl-change`/`sl-input` for Shoelace)
- store: VariableStore reference for ui-code execution scope
- unbindValue: Callback to stop watching the child variable
- unbindError: Callback to stop watching errors on the child variable

### Does (implemented by BindingEngine)
- createChildVariable: Create child variable with path property for backend resolution
- watchChildVariable: Watch the child variable (not parent) for value updates
- apply: Set element property from child variable value (uses defaultValue for nullish)
- update: Refresh element when child variable changes (handles nullish gracefully)
- getElement: Look up DOM element by elementId (via document.getElementById)
- getTargetProperty: Determine which element property to set
- transformValue: Apply any value transformations
- handleNullishRead: Display defaultValue when path resolves to null/undefined
- handleNullishWrite: Send error message with code 'path-failure' when write path is nullish (causes UI error indicator)
- scrollToBottom: Scroll element to bottom if `scrollOnOutput` option is set and element is scrollable
- selectUpdateEvent: Choose update event based on element type and `keypress` option
- executeCode: For code bindings, execute JavaScript with element, value, variable, and store in scope
- shouldSuppressUpdate: Check if update should be skipped due to duplicate value (see Duplicate Update Suppression)

## Unbind Handler

Each value binding registers an unbind handler with the Widget that:
1. Stops watching the child variable (value and error callbacks)
2. Destroys the child variable

Called automatically when `widget.unbindAll()` is invoked.

## Child Variable Architecture

**Critical: ValueBinding creates and manages a child variable for server-side path resolution.**

Variable values are object references (`{"obj": 1}`), not actual data. The frontend cannot resolve paths client-side. Each ValueBinding:

1. **Creates** a child variable: `store.create({parentId: contextVarId, properties: {path: "fieldName"}})`
2. **Watches** the child variable for value updates (backend sends resolved values)
3. **Watches** the child variable for errors (e.g., `path-failure`)
4. **Destroys** the child variable when unbound

This applies to ALL binding types: ui-value, ui-attr-*, ui-class-*, ui-style-*

## Nullish Path Handling

ValueBinding implements nullish-safe read/write behavior:
- **Read (variable -> element):** When path resolves to null/undefined, displays defaultValue (no error)
- **Write (element -> variable):** When write path is nullish, sends `error(varId, 'path-failure', description)` message. UI shows error indicator (e.g., `ui-error` class on element). Error clears on successful update.

This enables bindings like `ui-value="selectedContact.firstName"` to work gracefully when `selectedContact` is null.
When user attempts to edit a field with a nullish path, the field shows an error indicator until the path becomes valid.

## Default Access Property

ValueBinding determines the default `access` property based on binding type and element:

| Binding Type | Element Type | Default Access |
|--------------|--------------|----------------|
| `value` | Interactive native (input, textarea, select) | `rw` |
| `value` | Interactive Shoelace (sl-input, sl-checkbox, etc.) | `rw` |
| `value` | Read-only Shoelace (sl-progress-bar, sl-qr-code, etc.) | `r` |
| `value` | Non-interactive (div, span, etc.) | `r` |
| `attr` | Any | `r` |
| `class` | Any | `r` |
| `style` | Any | `r` |
| `code` | Any | `r` |

**Read-only Shoelace components** (from viewdefs.md table):
- `sl-copy-button`, `sl-option`, `sl-progress-bar`, `sl-progress-ring`, `sl-qr-code`

These components have a `value` property but no user-editable input, so they default to `access=r`.

When creating the child variable, if no explicit `access` property is in pathOptions, the default is applied.

## Code Binding Execution

For `ui-code` bindings, the `executeCode` method:

1. Receives the code string from the child variable value
2. Looks up the element by elementId (not stored reference)
3. Creates a function with controlled scope: `new Function('element', 'value', 'variable', 'store', code)`
4. Calls the function with the element, current value, variable, and VariableStore
5. Catches and logs any execution errors (does not throw)

**Scope variables:**
- `element` - The bound DOM element (looked up by element ID)
- `value` - The new value from the variable
- `variable` - The variable for this binding (provides access to widget via properties)
- `store` - The VariableStore for accessing/creating other variables

**Example execution:**
```javascript
// ui-code="formatCode" where variable value is "element.innerHTML = marked(value)"
const element = document.getElementById(this.elementId);
const fn = new Function('element', 'value', 'variable', 'store', code);
fn(element, currentValue, this.childVariable, this.store);
```

**Why element lookup instead of stored reference:**
- Avoids memory leaks from DOM references in closures
- Element may be removed/replaced; lookup ensures current element
- Consistent with Widget pattern (element ID indirection)

## Input Update Event Selection

For two-way bound input elements, the update event is selected based on:

| Element Type | Default Event | With `keypress` Property |
|--------------|---------------|--------------------------|
| `<input>` | `blur` | `input` |
| `<textarea>` | `blur` | `input` |
| `<sl-input>` | `sl-change` | `sl-input` |
| `<sl-textarea>` | `sl-change` | `sl-input` |

The `keypress` property is parsed from the path (e.g., `name?keypress`) and defaults to `true` when specified without a value.

## Local Value Setting (Universal Principle)

**Spec: viewdefs.md "Frontend Update Behavior"**

**Whenever the frontend sends a variable update to the backend, it MUST first set the value in the local variable cache.** This applies when ValueBinding sends updates in response to user input (blur/input events).

When the user changes an input value:
1. Set `variable.value` locally (MUST do this first)
2. Send update message to backend

This ensures the frontend's cached variable state accurately reflects the UI state being sent to the backend.

## Duplicate Update Suppression

**Spec: viewdefs.md "Frontend Update Behavior"**

Bindings that do NOT have `access=action` or `access=w` MUST NOT send an update if the variable's value has not changed.

**When to suppress:**
- Before sending an update, compare the new value to the variable's current cached value
- If values are equal, skip the update entirely (do not set local value, do not send message)

**When NOT to suppress (always send):**
- `access=action` - Actions are side-effect triggers, always send
- `access=w` - Write-only bindings, always send

**Implementation:**
```typescript
// shouldSuppressUpdate(variable, newValue): boolean
if (pathOptions.access === 'action' || pathOptions.access === 'w') {
  return false;  // Never suppress actions or write-only
}
return variable.value === newValue;  // Suppress if unchanged
```

**Rationale:**
- Reduces unnecessary network traffic
- Prevents redundant backend processing
- Blur events may fire without actual value changes
- Action bindings intentionally trigger side effects regardless of value

## Auto-Scroll on Output

The `scrollOnOutput` path property enables automatic scrolling to the bottom when a value updates:

```html
<div ui-value="log?scrollOnOutput"></div>
<pre ui-value="terminal?scrollOnOutput"></pre>
```

**Behavior:**
1. When the child variable receives an update (value change from backend)
2. After applying the value to the element
3. If `pathOptions.scrollOnOutput` is true:
   - Check if element is scrollable (`element.scrollHeight > element.clientHeight`)
   - If scrollable, set `element.scrollTop = element.scrollHeight`

**Use cases:**
- Log viewers showing streaming output
- Chat windows with new messages
- Terminal emulators with command output
- Any container displaying appended content

**Implementation:**
```typescript
// In update handler, after applying value
if (pathOptions.scrollOnOutput) {
  const element = document.getElementById(this.elementId);
  if (element && element.scrollHeight > element.clientHeight) {
    element.scrollTop = element.scrollHeight;
  }
}
```

**Notes:**
- Only scrolls if the element has overflow (is actually scrollable)
- Scroll happens after DOM update to ensure accurate scrollHeight
- Works with any element that can have overflow (div, pre, textarea, etc.)

## Parent Scroll Notifications

When `ui-value` updates an element, it may trigger scrolling on an ancestor widget with `scrollOnOutput`. This depends on whether the element resizes when its content changes.

**Content-resizable elements** (trigger parent scroll):
- `<span>`, `<div>`, `<p>`, `<pre>`, `<label>`, etc.
- These elements resize when their text content changes
- After applying the value, call `bindingEngine.addScrollNotification(parentVarId)`

**Fixed-size input elements** (do NOT trigger parent scroll):
- `<input>`, `<textarea>`, `<sl-input>`, `<sl-textarea>`
- These have fixed dimensions regardless of content value
- Do not add scroll notification after value update

**Detection:**
```typescript
const NON_RESIZING_ELEMENTS = new Set(['input', 'textarea', 'sl-input', 'sl-textarea']);

function triggersParentScroll(element: Element): boolean {
  return !NON_RESIZING_ELEMENTS.has(element.tagName.toLowerCase());
}
```

**Implementation in update handler:**
```typescript
// After applying value to element
if (triggersParentScroll(element)) {
  bindingEngine.addScrollNotification(this.childVariable.parentId);
}
```

**Rationale:**
- Content-resizable elements may grow/shrink, potentially pushing content below the viewport
- An ancestor scroll container with `scrollOnOutput` should scroll to show new content
- Input elements don't resize, so updating them doesn't affect layout

## ui-keypress Attribute

The `ui-keypress` attribute is a shorthand for `ui-value` with the `keypress` option:

```html
<input ui-keypress="name">
<!-- Equivalent to: <input ui-value="name?keypress"> -->
```

When `ui-keypress` is used, `pathOptions.keypress` is implicitly set to `true`, causing the binding to use keystroke events instead of blur events. All other behavior (child variable creation, path resolution, nullish handling) is identical to `ui-value`.

## Collaborators

- Widget: Binding context (provides element ID, variable mappings)
- BindingEngine: Creates and manages bindings
- Variable: Source of bound value
- VariableStore: Passed to ui-code execution scope, provides watch() for value change notifications
- WidgetBinder: Handles widget-specific bindings
- PathSyntax: Parses path options including `keypress`

## Sequences

- seq-bind-element.md: Creating value binding
- seq-update-variable.md: Propagating value changes
- seq-input-value-binding.md: Input element event selection and two-way binding
