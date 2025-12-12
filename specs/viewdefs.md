# Viewdef Binding

Viewdefs (HTML view definitions) can reference variables using binding syntax.

## View Definitions

View definitions ("viewdefs") are HTML snippets named `TYPE.NAMESPACE.html` (e.g., `Person.DEFAULT.html`, `Person.COMPACT.html`). They are stored in `html/viewdefs/` and delivered to the frontend via variable `1`.

**Viewdef format:**

Each viewdef is a string containing a single `<template>` element:

```html
<template>
  <div class="person-card">
    <span ui-value="name"></span>
    <span ui-value="email"></span>
  </div>
</template>
```

The frontend parses viewdefs by placing them in a scratch div's innerHTML, then validates:
- Exactly one root element exists
- The root element is a `<template>`
- If validation fails, sends an `error` message to the backend

**Bootstrap process:**
1. When a frontend connects, it immediately watches variable `1` (the only variable at startup)
2. Variable `1` contains the root object of the application
3. Variable `1` has a `viewdefs` property containing `TYPE.NAMESPACE` → `HTML` mappings
4. The frontend parses the viewdefs and stores them by TYPE.NAMESPACE

**Viewdef delivery:**
- When a variable is created or its value changes, the backend sets the `type` property based on the value's type
- If viewdefs for that type haven't been sent, the backend queues them
- Before sending updates, pending viewdefs are set on variable `1`'s `viewdefs` property with `:high` priority
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

## Action Bindings

- `ui-action` - Bind a button/element click to a method call on the presenter
  - Value is a path ending in a method call
  - `method()` - call with no arguments
  - `method(_)` - call with the update message's value as the argument
  - Examples:
    - `<sl-button ui-action="presenter.save()">Save</sl-button>` - calls `save()` with no args
    - `<sl-button ui-action="delegate.run(_)">Run</sl-button>` - calls `run(value)` with the update value

## Backend Paths

The `ui-*` attribute values can contain **paths** that navigate the presenter's data structure. Paths use dot notation to traverse nested objects.

**Examples:**
- `ui-value="name"` - Binds to the `name` field of the presenter
- `ui-value="father.name"` - Binds to the `name` field of the presenter's `father` object
- `ui-value="addresses.0.city"` - Binds to the first address's city (array index)
- `ui-value="getName()"` - Binds to a call on the persenter's `getName` method

**Nullish path handling:**

Path traversal uses nullish coalescing behavior (like JavaScript's `?.` operator). If any segment in the path resolves to `null` or `undefined`:
- **Read direction:** The binding displays empty/default value instead of erroring
- **Write direction:** The variable holder issues an `error` message with code `path-failure`, allowing the frontend to display an error state (e.g., red border on the field). A subsequent successful update clears the error condition.

This allows bindings like `ui-value="selectedContact.firstName"` to work gracefully when `selectedContact` is null (e.g., when no contact is selected). When a user attempts to edit a field with a nullish path, the field can show an error indicator until the path becomes valid.

Paths are stored in the variable's "path" property, allowing the backend to:
- Resolve the path to the actual data location in the presenter data
- Update the correct nested field when the variable changes
- Watch specific sub-paths for changes

## App View

The **App View** is a special view that renders variable `1` (the root app variable). It is the entry point for the entire UI.

**App View attribute:**
- `ui-app` - Marks an element as the app view container (renders variable `1`)

**App flow:**
1. Client connects to the server
2. Server sends an `update` message for variable `1` with app state and viewdefs
3. The `ui-app` element renders its view based on variable `1`'s `type` property

**Example (minimal index.html):**
```html
<!DOCTYPE html>
<html>
<head>
  <script type="module" src="main.js"></script>
</head>
<body>
  <div ui-app></div>
</body>
</html>
```

Developers can customize `index.html` with their own styles, scripts, and structure while keeping the `ui-app` element as the rendering root.

## Views

A **View** is an element that renders a variable using a viewdef. Views are created via the `ui-view` attribute.

**View attributes:**
- `ui-view` - Path to an object reference variable to render
- `ui-namespace` - (optional) Namespace for viewdef lookup (default: `DEFAULT`)

**View properties:**
- Unique HTML `id` (vended by the frontend)
- Manages a container element (e.g., `<div>`, `<sl-option>`)

**Example:**
```html
<div ui-view="currentContact" ui-namespace="COMPACT"></div>
```

This creates a variable for the `currentContact` object reference and renders it using the `TYPE.COMPACT` viewdef (where TYPE is the variable's `type` property).

## Rendering

The frontend maintains a `render(element, variable, namespace)` function:

**Parameters:**
- `element` - The container element to render into
- `variable` - The variable to render
- `namespace` - The viewdef namespace (default: `DEFAULT`)

**Returns:** `true` if rendered successfully, `false` if not ready

**Requirements for rendering:**
1. Variable must have a `value`
2. Variable must have a `type` property
3. Viewdef for `TYPE.NAMESPACE` must exist (falls back to `TYPE.DEFAULT` if not found)

**Render process:**
1. Look up viewdef for `TYPE.NAMESPACE`
2. If not found, try `TYPE.DEFAULT`
3. Clear the element's children
4. Deep clone the template's contents into the element
5. Bind the cloned elements to the variable

**Pending views:**

If a view cannot render (missing `type` or viewdef), it's added to a pending views list. When new viewdefs arrive (via variable `1` update):
1. Store the new viewdefs
2. Iterate pending views, calling `render()` on each
3. Remove views that return `true` (successfully rendered)
4. Views that return `false` remain pending

## ViewLists

A **ViewList** renders an array of object references as a list of views. Created via the `ui-viewlist` attribute.

**ViewList attributes:**
- `ui-viewlist` - Path to an array of object references (supports path properties)
- `ui-namespace` - (optional) Namespace for child views (default: `list-item`)

**Path properties for ViewList:**
```html
<!-- Basic ViewList -->
<div ui-viewlist="contacts">

<!-- With item presenter wrapper -->
<div ui-viewlist="contacts?item=ContactPresenter">
```

The `?item=PresenterType` property tells ViewList which presenter type to wrap each domain object with.

**ViewList as wrapper object:**

ViewList is a wrapper type (see protocol.md). When `ui-viewlist="path"` is used:
1. Frontend creates a variable with `wrapper=ViewList` in path properties
2. Backend instantiates a ViewList wrapper object: `ViewList(variable)`
3. The wrapper is stored internally in the variable
4. When the monitored value (domain array) changes, `viewList.computeValue(rawArray)` is called
5. ViewList maintains a parallel array of presenter objects and returns presenter refs as the stored value

The ViewList can access path properties like `item=ContactPresenter` from the variable's properties.

**ViewItem objects:**

ViewList creates a ViewItem object for each array element. Each ViewItem has:
- `baseItem` - Reference to the domain object (`{obj: ID}`)
- `item` - Either same as `baseItem`, or if `item=ItemWrapper` property is set, the result of `ItemWrapper(viewItem)`
- `list` - Reference to the ViewList object
- `index` - Position in the list (0-based)

**Item wrapping (optional):**

When `item=PresenterType` is specified in path properties, the ViewItem's `item` property holds a wrapped presenter instead of the raw domain object. The ItemWrapper is constructed with the ViewItem: `ItemWrapper(viewItem)`.

This allows presenters to have UI-specific methods like `delete()` that can:
- Access the domain object via `viewItem.baseItem`
- Remove itself via `viewItem.list.removeAt(viewItem.index)`

**ViewItem viewdef:**

ViewList uses the `list-item` namespace by default for its ViewItems. The ViewItem's `list-item` viewdef contains a view on `item` that also uses the `list-item` namespace, plus a delete button:

```html
<!-- ViewItem.list-item viewdef -->
<template>
  <div style="display: flex; align-items: center;">
    <div ui-view="item" ui-namespace="list-item" style="flex: 1;"></div>
    <sl-icon-button ui-action="remove()" name="x" label="Remove"></sl-icon-button>
  </div>
</template>
```

Developers can specify a custom `ui-namespace` on the ViewList to use a different ViewItem viewdef (e.g., one without the delete button):

```html
<!-- ViewItem.readonly viewdef (no delete button) -->
<template>
  <div ui-view="item" ui-namespace="list-item"></div>
</template>
```

**ViewList properties:**
- Has an **exemplar element** that gets cloned for each item (default: `<div>`)
- Maintains a parallel array of View elements (these don't have `ui-view` attributes - the ViewList creates their variables)
- Has a delegate for add/remove notifications

**Example:**
```html
<div ui-viewlist="contacts" ui-namespace="COMPACT"></div>
```

**Update behavior:**

When the bound array changes:
- **Items added:** Clone the exemplar, create a variable for the new item, render and append
- **Items removed:** Destroy the variable, remove the element from DOM
- **Items reordered:** Reorder the parallel View elements to match

## Select Views

A **Select View** uses a ViewList to populate `<sl-select>` options.

**Example:**
```html
<sl-select ui-value="selectedContact">
  <div ui-viewlist="contacts" ui-namespace="OPTION"></div>
</sl-select>
```

The Select View provides `<sl-option>` as the exemplar element to its ViewList, so each item renders as an option:

```html
<!-- Contact.OPTION.html viewdef -->
<template>
  <sl-option ui-attr-value="id">
    <span ui-value="name"></span>
  </sl-option>
</template>
```
