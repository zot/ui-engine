# Viewdef Binding

Viewdefs (HTML view definitions) can reference variables using binding syntax.

## View Definitions

View definitions ("viewdefs") are HTML snippets named `TYPE.VIEW.html` (e.g., `Person.DEFAULT.html`, `Person.list-item.html`). They are delivered to the frontend via variable `1`.

**Bootstrap process:**
1. When a frontend connects, it immediately watches variable `1` (the only variable at startup)
2. Variable `1` contains the root object of the application
3. Variable `1` has a `viewdefs` property containing `TYPE.VIEW` → `HTML` mappings
4. The frontend parses the viewdefs and stores them by TYPE.VIEW

**Viewdef delivery:**
- When a variable's type changes, the backend sets the `type` property and includes the viewdefs for that type in variable `1`'s `viewdefs` property
- The backend accumulates viewdef updates for batching and prioritizes the update message
- Previous `viewdefs` property values can be safely replaced since the frontend stores viewdefs separately

## Value Bindings (variable → element)

- `ui-value` - Bind a variable to the element's "value" (input field, file name, etc.)
  - The attribute value becomes the variable's "path" property for backend use
- `ui-attr-*` - Bind a variable value to an HTML attribute (e.g., `ui-attr-disabled`)
- `ui-class-*` - Bind a variable value to CSS classes (value is a class string)
- `ui-style-*-*` - Bind a variable value to a CSS style property

Variable values are used directly; variable properties can specify transformations.

## Event Bindings (element → variable)

- `ui-event-*` - Trigger a variable value change on an event (e.g., `ui-event-click`, `ui-event-change`)
  - Sets the bound variable to a specified value when the event fires

## Backend Paths

The `ui-*` attribute values can contain **paths** that navigate the presenter's data structure. Paths use dot notation to traverse nested objects.

**Examples:**
- `ui-value="name"` - Binds to the `name` field of the presenter
- `ui-value="father.name"` - Binds to the `name` field of the presenter's `father` object
- `ui-value="addresses.0.city"` - Binds to the first address's city (array index)
- `ui-value="getName()"` - Binds to a call on the persenter's `getName` method

Paths are stored in the variable's "path" property, allowing the backend to:
- Resolve the path to the actual data location in the presenter data
- Update the correct nested field when the variable changes
- Watch specific sub-paths for changes
