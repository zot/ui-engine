# ValueBinding

**Source Spec:** viewdefs.md, libraries.md

## Responsibilities

### Knows
- element: Target DOM element
- childVarId: ID of the child variable created for this binding (NOT the parent context variable)
- bindingType: One of value, keypress, attr, class, style, code
- attributeName: For attr/class/style, the specific attribute
- path: Path property value sent to backend for resolution
- pathOptions: Parsed path options including `keypress`, `create`, `wrapper`, `access`, etc.
- defaultValue: Empty/default value for nullish paths (empty string, false, etc.)
- updateEvent: Event to listen for updates (`blur`/`input` for native, `sl-change`/`sl-input` for Shoelace)
- unbindValue: Callback to stop watching the child variable
- unbindError: Callback to stop watching errors on the child variable

### Does
- createChildVariable: Create child variable with path property for backend resolution
- watchChildVariable: Watch the child variable (not parent) for value updates
- apply: Set element property from child variable value (uses defaultValue for nullish)
- update: Refresh element when child variable changes (handles nullish gracefully)
- getTargetProperty: Determine which element property to set
- transformValue: Apply any value transformations
- destroy: Clean up binding, unwatch, and **destroy child variable**
- handleNullishRead: Display defaultValue when path resolves to null/undefined
- handleNullishWrite: Send error message with code 'path-failure' when write path is nullish (causes UI error indicator)
- selectUpdateEvent: Choose update event based on element type and `keypress` option
- executeCode: For code bindings, execute JavaScript with element and value in scope

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
| `value` | Interactive (input, textarea, select, sl-*) | `rw` |
| `value` | Non-interactive (div, span, etc.) | `r` |
| `attr` | Any | `r` |
| `class` | Any | `r` |
| `style` | Any | `r` |
| `code` | Any | `r` |

When creating the child variable, if no explicit `access` property is in pathOptions, the default is applied.

## Code Binding Execution

For `ui-code` bindings, the `executeCode` method:

1. Receives the code string from the child variable value
2. Creates a function with controlled scope: `new Function('element', 'value', code)`
3. Calls the function with the bound element and current value
4. Catches and logs any execution errors (does not throw)

**Example execution:**
```javascript
// ui-code="formatCode" where variable value is "element.innerHTML = marked(value)"
const fn = new Function('element', 'value', code);
fn(this.element, currentValue);
```

## Input Update Event Selection

For two-way bound input elements, the update event is selected based on:

| Element Type | Default Event | With `keypress` Property |
|--------------|---------------|--------------------------|
| `<input>` | `blur` | `input` |
| `<textarea>` | `blur` | `input` |
| `<sl-input>` | `sl-change` | `sl-input` |
| `<sl-textarea>` | `sl-change` | `sl-input` |

The `keypress` property is parsed from the path (e.g., `name?keypress`) and defaults to `true` when specified without a value.

## ui-keypress Attribute

The `ui-keypress` attribute is a shorthand for `ui-value` with the `keypress` option:

```html
<input ui-keypress="name">
<!-- Equivalent to: <input ui-value="name?keypress"> -->
```

When `ui-keypress` is used, `pathOptions.keypress` is implicitly set to `true`, causing the binding to use keystroke events instead of blur events. All other behavior (child variable creation, path resolution, nullish handling) is identical to `ui-value`.

## Collaborators

- BindingEngine: Creates and manages bindings
- Variable: Source of bound value
- WatchManager: Notifies of value changes
- WidgetBinder: Handles widget-specific bindings
- PathSyntax: Parses path options including `keypress`

## Sequences

- seq-bind-element.md: Creating value binding
- seq-update-variable.md: Propagating value changes
- seq-input-value-binding.md: Input element event selection and two-way binding
