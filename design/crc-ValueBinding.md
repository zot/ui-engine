# ValueBinding

**Source Spec:** viewdefs.md

## Responsibilities

### Knows
- element: Target DOM element
- variableId: Bound variable ID
- bindingType: One of value, attr, class, style
- attributeName: For attr/class/style, the specific attribute
- path: Path property value for backend use
- defaultValue: Empty/default value for nullish paths (empty string, false, etc.)

### Does
- apply: Set element property from variable value (uses defaultValue for nullish)
- update: Refresh element when variable changes (handles nullish gracefully)
- getTargetProperty: Determine which element property to set
- transformValue: Apply any value transformations
- destroy: Clean up binding and unwatch variable
- handleNullishRead: Display defaultValue when path resolves to null/undefined
- handleNullishWrite: Send error message with code 'path-failure' when write path is nullish (causes UI error indicator)

## Nullish Path Handling

ValueBinding implements nullish-safe read/write behavior:
- **Read (variable -> element):** When path resolves to null/undefined, displays defaultValue (no error)
- **Write (element -> variable):** When write path is nullish, sends `error(varId, 'path-failure', description)` message. UI shows error indicator (e.g., `ui-error` class on element). Error clears on successful update.

This enables bindings like `ui-value="selectedContact.firstName"` to work gracefully when `selectedContact` is null.
When user attempts to edit a field with a nullish path, the field shows an error indicator until the path becomes valid.

## Collaborators

- BindingEngine: Creates and manages bindings
- Variable: Source of bound value
- WatchManager: Notifies of value changes
- WidgetBinder: Handles widget-specific bindings

## Sequences

- seq-bind-element.md: Creating value binding
- seq-update-variable.md: Propagating value changes
