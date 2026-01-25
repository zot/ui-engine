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

**Hot-reloading:**

Viewdefs support hot-reloading for iterative development. See [Hot-Loading System](main.md#hot-loading-system) for the unified backend behavior (file watching, symlink tracking, session refresh).

**Additional viewdef behavior:**
- On-demand loading: when a new `type` is encountered, the server automatically loads matching `TYPE.*.html` files from the viewdef directory
- This enables editing viewdefs without restarting the server

**Frontend behavior:**
- Views are marked with a `ui-viewdef` attribute containing the viewdef key (e.g., `ui-viewdef="Contact.COMPACT"`)
- Widgets store a reference to their containing view's element ID (if any)
- When new viewdefs arrive via variable 1:
  1. Store the updated viewdefs
  2. For each updated viewdef key, find all views with matching `ui-viewdef`
  3. Re-render each matching view using the updated viewdef
  4. Re-binding occurs automatically as part of the render process

**Variable destruction on re-render:**

When a View or ViewList is destroyed (during hot-reload re-render or explicit destruction), it must destroy its associated variable. This is critical for proper resource cleanup:
- Child views/viewlists create variables via `VariableStore.create()`
- When destroyed, they must call `VariableStore.destroy(varId)` to clean up
- Backend destruction is recursive - destroying a parent variable also destroys children
- This prevents variable leaks during hot-reload cycles

## Element References (Cross-Cutting Requirement)

**Frontend code MUST NOT store direct references to DOM elements.**

Instead, all element references must be stored as element IDs. When an element needs to be accessed, look it up by ID using `document.getElementById(elementId)`.

**Global Element ID Vendor:**
- A single global counter starts at `1` and increments
- Format: `ui-{counter}` (e.g., `ui-1`, `ui-2`, `ui-3`)
- Used by: Widgets, Views, ViewLists, or any code that needs a unique element ID
- When an element doesn't have an ID, the vendor assigns one

**Rationale:**
- Avoids circular references and memory leaks from DOM element references
- Enables serialization of binding/widget state
- Simplifies garbage collection
- Elements can be looked up on demand via `document.getElementById()`

**Implementation pattern:**
```typescript
// ❌ WRONG - storing element reference
private element: Element

// ✅ CORRECT - storing element ID
private elementId: string

// Access element when needed
const element = document.getElementById(this.elementId)
```

## Widgets

A **Widget** is the binding context for an element with `ui-*` bindings. Each element with bindings has exactly one Widget that manages all its bindings.

**All bindings create widgets:** Every binding type (`ui-value`, `ui-attr-*`, `ui-view`, `ui-viewlist`, etc.) creates and registers a Widget. This is necessary because any element could become a scroll container via CSS, and scroll-related behavior (like `scrollOnOutput`) is managed at the Widget level.

**Widget properties:**
- `elementId` - Element ID (from global vendor if element has no ID)
- `variables` - Map of binding name to variable ID for all bindings on this element
- `unbindHandlers` - Map of binding name to cleanup function
- `scrollOnOutput` - If true, scroll element to bottom when child content renders (set via path property)

**Widget responsibilities:**
- Tracks all bindings and their variable IDs for the element
- Provides cleanup via `unbindAll()` which calls all unbind handlers
- Enables bindings to look up sibling bindings (e.g., event binding finding ui-value variable)
- Manages scroll behavior via `scrollOnOutput` property and `scrollToBottom()` method

**Element ID assignment:**
- If an element doesn't have an ID, the Widget uses the global ID vendor to assign one

**Variable-Widget relationship:**
- Variables do NOT store direct references to DOM elements (use element ID lookup instead)
- This enables proper cleanup and avoids memory leaks from DOM references

## Value Bindings (variable → element)

- `ui-value` - Bind a variable to the element's "value" (input field, file name, etc.)
  - For non-interactive elements (div, span, etc.), automatically adds `access=r` if no `access` property is specified
  - Interactive elements (input, textarea, select, etc.) default to read-write
  - Shoelace components (`sl-*`): defaults depend on the component's read-only status (see table below)
    - Components with `read-only=yes` default to `access=r`
    - Other `ui-value` components default to read-write
    - Explicit `access` in the path always overrides the default
  - Additional path property: `keypress` (see Path Properties section)

**Shoelace Component Interactivity:**

| Component        | property attribute | read-only | Notes                                            |
|------------------|:------------------:|-----------|--------------------------------------------------|
| sl-input         | ui-value           |           | Text input with change/input events              |
| sl-textarea      | ui-value           |           | Multi-line text input                            |
| sl-select        | ui-value           |           | Dropdown selection                               |
| sl-checkbox      | ui-value           |           | Boolean toggle                                   |
| sl-radio         | ui-value           |           | Single selection (use within sl-radio-group)     |
| sl-radio-group   | ui-value           |           | Container for radio buttons                      |
| sl-radio-button  | ui-value           |           | Styled radio option                              |
| sl-switch        | ui-value           |           | Toggle switch                                    |
| sl-range         | ui-value           |           | Slider input                                     |
| sl-color-picker  | ui-value           |           | Color selection                                  |
| sl-rating        | ui-value           |           | Star rating input                                |
| sl-button        | ui-action          |           | Use `ui-action` instead                          |
| sl-icon-button   | ui-action          |           | Use `ui-action` instead                          |
| sl-copy-button   | ui-value           | yes       | Has internal copy behavior                       |
| sl-details       | none               |           | Expandable panel (has toggle event but no value) |
| sl-dialog        | none               |           | Modal dialog                                     |
| sl-drawer        | none               |           | Slide-out panel                                  |
| sl-dropdown      | none               |           | Popup container                                  |
| sl-menu          | none               |           | Menu container                                   |
| sl-menu-item     | ui-action          |           | Use `ui-action` for selection                    |
| sl-tab-group     | none               |           | Tab container (has show event but no value)      |
| sl-tab           | none               |           | Tab header                                       |
| sl-tree          | none               |           | Tree container                                   |
| sl-tree-item     | none               |           | Tree node                                        |
| sl-alert         | none               |           | Notification (closable but no value)             |
| sl-avatar        | none               |           | Display only                                     |
| sl-badge         | none               |           | Display only; bind to child element              |
| sl-breadcrumb    | none               |           | Display only                                     |
| sl-card          | none               |           | Display only                                     |
| sl-carousel      | none               |           | Display only                                     |
| sl-divider       | none               |           | Display only                                     |
| sl-format-bytes  | none               |           | Display only                                     |
| sl-format-date   | none               |           | Display only                                     |
| sl-format-number | none               |           | Display only                                     |
| sl-icon          | none               |           | Display only                                     |
| sl-option        | ui-value           | yes       | Used inside sl-select                            |
| sl-progress-bar  | ui-value           | yes       | Display only                                     |
| sl-progress-ring | ui-value           | yes       | Display only                                     |
| sl-qr-code       | ui-value           | yes       | Display only                                     |
| sl-relative-time | none               |           | Display only                                     |
| sl-skeleton      | none               |           | Display only                                     |
| sl-spinner       | none               |           | Display only                                     |
| sl-tag           | none               |           | Display only                                     |
| sl-tooltip       | none               |           | Display only                                     |

**Shoelace tips:**
- For non-interactive components, bind to a child element: `<sl-badge><span ui-value="count"></span></sl-badge>`
- Use `ui-action` for button clicks: `<sl-button ui-action="save()">Save</sl-button>`
- `ui-attr-*` - Bind a variable value to an HTML attribute (e.g., `ui-attr-disabled`); defaults to `access=r`
- `ui-class-*` - Bind a variable value to CSS classes (value is a class string); defaults to `access=r`
- `ui-style-*` - Bind a variable value to a CSS style property (e.g., `ui-style-background-color`); defaults to `access=r`
- `ui-code` - Execute JavaScript code when the variable receives an update; defaults to `access=r`
  - The attribute value is a path to a variable containing JS code
  - When the variable's value changes, the code is executed
  - The code has access to:
    - `element` - The bound element (looked up by element ID, not a stored reference)
    - `value` - The new value from the variable
    - `variable` - The variable for this binding (provides access to widget via properties)
    - `store` - The VariableStore for accessing/creating other variables
- `ui-html` - Bind a variable to the element's innerHTML; defaults to `access=r`
  - The attribute value is a path to a variable containing HTML markup
  - When the variable's value changes, the HTML is rendered into the element
  - Additional path property: `replace` (see below)

**ui-html replace mode:**

When `replace` is specified in the path (e.g., `ui-html="content?replace"`), the element is replaced with the HTML content instead of setting innerHTML:

1. **ID preservation:** The first element in the HTML content receives the original view element's ID (since the original element will be removed from the DOM)

2. **Fragment handling:** If the HTML produces multiple elements (a fragment):
   - The first element gets the view element's original ID
   - Subsequent elements get IDs from the global ID vendor
   - The widget tracks the complete list of element IDs, not just the first one

3. **Update behavior:** When the HTML content changes:
   - All tracked elements (the entire fragment) are removed from the DOM
   - The new HTML is inserted at the position of the first tracked element
   - IDs are reassigned following the same rules (first gets original ID, rest get vended IDs)

4. **Cleanup:** When the widget is unbound, all tracked elements are removed

**Example:**
```html
<!-- Standard innerHTML mode -->
<div ui-html="description"></div>

<!-- Replace mode - element replaced with HTML content -->
<div ui-html="renderedMarkdown?replace"></div>
```

Variable values are used directly; variable properties can specify transformations.

### Path Resolution: Server-Side Only

**Critical: All path-based bindings MUST create child variables for backend path resolution.**

Variable values sent to the frontend are **object references** (e.g., `{"obj": 1}`), not actual object contents. Object references are stable identifiers that only change when the variable points to a different object entirely. This means:

- The frontend receives `{"obj": 1}` for a view variable, not `{name: "...", isActive: false, ...}`
- Client-side path resolution (extracting `isActive` from `{"obj": 1}`) is **impossible**
- All paths must be resolved by the backend, which has access to the actual object data

**Implementation requirement:**

Every binding that uses a path (`ui-value`, `ui-attr-*`, `ui-class-*`, `ui-style-*`) must:

1. Parse the path from the attribute value
2. Create a **child variable** under the context variable with `path` property set
3. Watch the **child variable** (not the parent) for value updates
4. The backend resolves the path and sends the actual value (boolean, string, number, etc.)
5. Destroy the child variable when the binding is unbound

**Example:**

```html
<div ui-attr-hidden="isEditView">
```

The binding engine must:
1. Create child variable: `{parentId: contextVarId, properties: {path: "isEditView"}}`
2. Watch the child variable for updates
3. Backend resolves `isEditView` on the parent object and sends `true` or `false`
4. Binding updates the `hidden` attribute based on the boolean value

## Frontend Update Behavior

**Whenever the frontend sends a variable update to the backend, it MUST first set the value in the local variable cache.** This ensures the frontend's cached variable state accurately reflects the UI state being sent to the backend.

**Duplicate update suppression:** Bindings that do NOT have `access=action` or `access=w` MUST NOT send an update if the variable's value has not changed. Before sending an update, compare the new value to the variable's current cached value; if they are equal, skip the update.

## Event Bindings (element → variable)

- `ui-event-*` - Trigger a variable value change on an event (e.g., `ui-event-click`, `ui-event-change`)
  - Sets the bound variable to a specified value when the event fires
  - **Value sync with `ui-value`:** When an event fires on an element that also has a `ui-value` binding:
    1. Check if the element's current value differs from the variable's cached value
    2. If different, send a variable update with the new value first
    3. Then send the event update
- `ui-event-keypress-*` - Trigger on specific key presses (e.g., `ui-event-keypress-enter`, `ui-event-keypress-ctrl-enter`)
  - Fires only when the specified key (and modifiers, if any) is pressed
  - **Format:** `ui-event-keypress-{modifiers}-{key}` where modifiers are optional
  - **Modifiers** (can be combined in any order before the key):
    - `ctrl` - Control key must be held
    - `shift` - Shift key must be held
    - `alt` - Alt key must be held
    - `meta` - Meta/Command key must be held
  - **Examples with modifiers:**
    - `ui-event-keypress-ctrl-enter` - Ctrl+Enter
    - `ui-event-keypress-shift-enter` - Shift+Enter
    - `ui-event-keypress-ctrl-shift-s` - Ctrl+Shift+S
    - `ui-event-keypress-alt-left` - Alt+Left arrow
  - Key names are case-insensitive and match keyboard event `key` values:
    - `ui-event-keypress-enter` - Enter/Return key
    - `ui-event-keypress-escape` - Escape key
    - `ui-event-keypress-left` - Left arrow key
    - `ui-event-keypress-right` - Right arrow key
    - `ui-event-keypress-up` - Up arrow key
    - `ui-event-keypress-down` - Down arrow key
    - `ui-event-keypress-tab` - Tab key
    - `ui-event-keypress-space` - Space bar
    - `ui-event-keypress-{letter}` - Any single letter (e.g., `ui-event-keypress-a`)
  - Listens on the `keydown` event of the element
  - **Modifier matching is exact:** If modifiers are specified, they must all be pressed and no additional modifiers should be pressed (e.g., `ui-event-keypress-ctrl-s` won't fire if Ctrl+Shift+S is pressed)
  - **Value behavior depends on path type:**
    - **Non-action path** (e.g., `lastKey`): Sets the variable to the lowercase key name (e.g., `"enter"`)
    - **No-arg action** (e.g., `selectFirst()`): Invokes the action with `null` (side-effect only)
    - **1-arg action** (e.g., `handleKey(_)`): Invokes the action with the key name as the argument

## Action Bindings

- `ui-action` - Bind a button/element click to a method call on the presenter
  - Value is a path ending in a method call; remember that paths can have properties
  - `method()` - call with no arguments
  - `method(_)` - call with the update message's value as the argument
  - Examples:
    - `<sl-button ui-action="presenter.save()">Save</sl-button>` - calls `save()` with no args
    - `<sl-button ui-action="delegate.run(_)">Run</sl-button>` - calls `run(value)` with the update value

## Backend Paths

**All `ui-*` binding attributes contain paths.** Paths navigate the presenter's data structure using dot notation to traverse nested objects.

**Examples:**
- `ui-value="name"` - Binds to the `name` field of the presenter
- `ui-value="father.name"` - Binds to the `name` field of the presenter's `father` object
- `ui-value="addresses.0.city"` - Binds to the first address's city (array index)
- `ui-value="getName()"` - Binds to a call on the presenter's `getName` method
- `ui-value="value()?access=rw"` - Read/write method (Lua only, see below)

**Nullish path handling:**

Path traversal uses nullish coalescing behavior (like JavaScript's `?.` operator). If any segment in the path resolves to `null` or `undefined`:
- **Read direction:** The binding displays empty/default value instead of erroring
- **Write direction:** The variable holder issues an `error` message with code `path-failure`, allowing the frontend to display an error state (e.g., red border on the field). A subsequent successful update clears the error condition.

This allows bindings like `ui-value="selectedContact.firstName"` to work gracefully when `selectedContact` is null (e.g., when no contact is selected). When a user attempts to edit a field with a nullish path, the field can show an error indicator until the path becomes valid.

Paths are stored in the variable's "path" property, allowing the backend to:
- Resolve the path to the actual data location in the presenter data
- Update the correct nested field when the variable changes
- Watch specific sub-paths for changes

## Path Properties

**All bindings use paths, and all paths can specify path properties.**

Path properties use URL-style query parameters appended to the path:

```
path.to.value?property1=value1&property2=value2
```

Properties without values default to `true`:
```
path?scrollOnOutput        <!-- equivalent to path?scrollOnOutput=true -->
```

**Universal path properties (supported by all bindings):**
- `scrollOnOutput` - Auto-scroll element to bottom when child content renders. Applies to the element's widget, not the variable, so it works with any binding type since any element could be a scroll container via CSS.
- `access` - Override default access mode (`r`, `rw`, `w`, `action`)

**Binding-specific path properties:**
- `keypress` - For input elements, send updates on every keypress instead of blur
- `wrapper` - Specify a wrapper type (e.g., `wrapper=lua.ViewList`)
- `item` - Specify item wrapper for lists (e.g., `item=ContactPresenter`)
- `create` - Create a new object of specified type
- `replace` - For `ui-html`, replace the element with the HTML content instead of setting innerHTML

**Examples across binding types:**
```html
<div ui-value="log?scrollOnOutput"></div>
<input ui-value="search?keypress">
<div ui-view="messages?scrollOnOutput"></div>
<div ui-viewlist="contacts?item=ContactPresenter"></div>
<div ui-attr-disabled="isLocked?scrollOnOutput"></div>
<div ui-class-active="isSelected?scrollOnOutput"></div>
<div ui-html="description"></div>
<div ui-html="renderedContent?replace"></div>
```

Path properties are parsed by the frontend and either:
- Handled locally (e.g., `scrollOnOutput` sets widget property)
- Sent to backend as variable properties (e.g., `wrapper`, `item`, `access`)

### Read/Write Methods (Lua Only)

Methods can be used as read/write properties by combining `()` syntax with `?access=rw`:

```html
<input ui-value="value()?access=rw">
```

**Behavior:**
- On **read**: The resolver calls the method with no arguments (just `self`)
- On **write**: The resolver calls the method with the value as an argument

This works because Lua supports varargs - the same function can handle both cases:

```lua
function MyPresenter:value(...)
    if select('#', ...) > 0 then
        self._value = select(1, ...)  -- write
    end
    return self._value  -- read
end
```

**Access mode constraints for methods:**
- `method()` with `access=r` or `access=action`: Read-only method call
- `method(_)` with `access=w` or `access=action`: Write-only method call (explicit argument)
- `method()` with `access=rw`: Read/write method (Lua only, argument is optional)

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
- `ui-namespace` - (optional) Namespace for viewdef lookup

The `ui-view` binding automatically adds `access=r` if no `access` property is specified, since views are typically read-only bindings. See Path Properties section for universal properties supported by all bindings.

**Namespace variable properties:**

When creating a view's variable:
- Find the closest element with `ui-namespace` using `element.closest('[ui-namespace]')`:
  - If found and either there's no parent variable or the parent variable's element contains it, use that namespace
  - Otherwise, inherit `namespace` property from the parent variable (if set)
- Inherit `fallbackNamespace` property from the parent variable (if set)

**Example:**

```html
<!-- Parent variable's element -->
<div ui-view="contact">
  <!-- Viewdef content -->
  <div ui-namespace="COMPACT">
    <div ui-view="address"></div>  <!-- inherits COMPACT from intermediate element -->
  </div>
</div>
```

The `address` view inherits `COMPACT` from the intermediate `<div>`, not from the parent variable's namespace property.

**Namespace resolution for rendering:**
1. If variable has `namespace` property and `TYPE.{namespace}` viewdef exists, use it
2. Otherwise, if variable has `fallbackNamespace` property and `TYPE.{fallbackNamespace}` viewdef exists, use it
3. Otherwise, use `TYPE.DEFAULT`

**View properties:**
- Unique HTML `id` (vended by the frontend)
- Manages a container element (e.g., `<div>`, `<sl-option>`)
- `ui-viewdef` attribute on the first element containing the viewdef key (e.g., `Contact.COMPACT`) for hot-reload targeting

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
3. A matching viewdef must exist (see namespace resolution below)

**Render process:**
1. Look up viewdef using namespace resolution:
   - If variable has `namespace` property and `TYPE.{namespace}` exists, use it
   - Else if variable has `fallbackNamespace` property and `TYPE.{fallbackNamespace}` exists, use it
   - Else use `TYPE.DEFAULT`
2. Set `ui-viewdef` attribute on the first element to the resolved viewdef key (e.g., `Contact.COMPACT`)
3. Clear the element's children (unbinding existing widgets)
4. Deep clone the template's contents (returns DocumentFragment, not yet in DOM)
5. Collect all `<script>` elements from the cloned content (store for later activation)
6. Append cloned nodes to the element (nodes are now in DOM)
7. Bind the cloned elements to the variable (widgets receive the view element ID)
8. Activate script elements (scripts are now DOM-connected):
   - For each collected script element:
     - Create a new `<script>` element via `document.createElement('script')`
     - Set `type` to `text/javascript`
     - Copy the `textContent` from the original to the new element
     - Replace the original script element with the new one

**Pending views:**

If a view cannot render (missing `type` or viewdef), it's added to a pending views list. When new viewdefs arrive (via variable `1` update):
1. Store the new viewdefs
2. Iterate pending views, calling `render()` on each
3. Remove views that return `true` (successfully rendered)
4. Views that return `false` remain pending

**Hot-reload re-rendering:**

When updated viewdefs arrive (viewdefs that were already sent):
1. For each updated viewdef key (e.g., `Contact.COMPACT`):
   - Query `document.querySelectorAll('[ui-viewdef="Contact.COMPACT"]')`
   - For each matching view element, re-render using the updated viewdef
2. Re-rendering reuses the same variable and container element
3. Widgets within the view are unbound and recreated during re-render

**Render notifications (for scrollOnOutput):**

When content changes may affect an element's size, it may need to trigger scrolling on an ancestor with `scrollOnOutput`. This is batched to avoid multiple scrolls during a single update cycle.

**Triggers for scroll notifications:**
- View or viewlist item renders
- `ui-value` updates a **content-resizable element** (span, div, p, etc.) - elements whose size changes when their text content changes

**Non-triggers:**
- `ui-value` updates an **input element** (input, textarea, sl-input, sl-textarea) - these have fixed dimensions regardless of content

1. **After content change:** The binding adds its parent variable ID to a global `pendingScrollNotifications` set
2. **After batch completes:** The BindingEngine processes using current/next pattern:
   - `current` = pendingScrollNotifications, `next` = empty set
   - While `current` is not empty:
     - For each variable ID in `current`:
       - Look up the widget for this variable ID
       - If the widget has `scrollOnOutput`:
         - Call the widget's `scrollToBottom()` (don't bubble further)
       - Otherwise, add the variable's parent ID to `next` (bubble up)
     - Clear `current`, swap `current` and `next`
   - Clear pendingScrollNotifications

This ensures:
- Multiple child renders in one batch cause only one scroll
- Scrolling happens at the correct ancestor widget (the one with `scrollOnOutput`)
- Views inside ViewLists trigger scrolling on the ViewList's widget or any ancestor with `scrollOnOutput`

## ViewLists

A **ViewList** renders an array of object references as a list of views.

**Standard list usage (recommended):**

Use `ui-view` with the `wrapper=lua.ViewList` path property:

```html
<!-- Basic list -->
<div ui-view="contacts?wrapper=lua.ViewList"></div>

<!-- With item presenter wrapper -->
<div ui-view="contacts?wrapper=lua.ViewList&itemWrapper=ContactPresenter"></div>
```

Additional path properties: `wrapper`, `item` (see Path Properties section)

**Alternative: ui-viewlist attribute:**

The `ui-viewlist` attribute is a shorthand that implicitly uses `lua.ViewList`:

```html
<div ui-viewlist="contacts"></div>
<!-- Equivalent to: <div ui-view="contacts?wrapper=lua.ViewList&access=r"></div> -->

<!-- With scrollOnOutput for auto-scrolling lists -->
<div ui-viewlist="messages?scrollOnOutput"></div>
```

The `ui-viewlist` binding automatically adds `access=r` if no `access` property is specified, since lists are typically read-only bindings. Additional path property: `item` (see Path Properties section for universal properties)

With `ui-viewlist`, use `ui-namespace` to specify the item viewdef namespace. Namespace inheritance follows the same rules as Views (see above).

**ViewList as wrapper object:**

ViewList is a wrapper type (see protocol.md). When `ui-viewlist="path"` is used:
1. Frontend creates a variable with `wrapper=lua.ViewList` in path properties.
2. The backend's `WrapperFactory` for `lua.ViewList` is called: `NewViewList(runtime, variable)`.
3. The `ViewList` wrapper stores the variable (the array is accessed via `variable:getValue()`).
4. The wrapper sets `fallbackNamespace:high` property to `list-item` on the variable (high priority ensures it's available before rendering).
5. The wrapper is registered in the object registry and stands in for child path navigation.
6. `ViewList` maintains:
   - `items` - array of `ViewListItem` objects, one per array element.
   - `selectionIndex` - current selection index for frontend use (default: 0 or -1 for no selection).

**Wrapper reuse and sync:** When the bound array changes, the `WrapperFactory` is called again. `ViewList` can return the existing wrapper and sync its `ViewListItems` with the new array:
1. For each item in the array, assign it to the corresponding ViewListItem's `item` property
2. If the ViewListItem array is longer than the item array, trim excess ViewListItems
3. If the ViewListItem array is shorter, create new ViewListItems for the additional items

This preserves internal state (like selection) while keeping ViewListItems in sync with the array.

The ViewList can access path properties like `itemWrapper=ContactPresenter` from the variable's properties.

**ViewListItem objects:**

ViewList creates a ViewListItem object for each array element. Each ViewListItem has:
- `item` - Pointer to the domain object from the array (resolved on the backend to a real object)
- `list` - Pointer to the ViewList object
- `index` - Position in the list (0-based)

**Custom ViewListItems (optional):**

When `item=PresenterType` is specified in path properties, ViewList creates instances of that type instead of plain ViewListItems. The custom type is constructed with the ViewList and index: `PresenterType:new(viewList, index)`.

This allows custom ViewListItems to have UI-specific methods like `delete()` that can:
- Access the domain object via `self.item`
- Remove itself via `self.list:removeAt(self.index)`

**ViewListItem viewdef:**

ViewList uses the `list-item` namespace by default for its ViewListItems. A typical viewdef renders the `item` (the domain object) with a delete button:

```html
<!-- ViewListItem.list-item viewdef -->
<template>
  <div style="display: flex; align-items: center;">
    <div ui-view="item" ui-namespace="list-item" style="flex: 1;"></div>
    <sl-icon-button ui-action="remove()" name="x" label="Remove"></sl-icon-button>
  </div>
</template>
```

Developers can specify a custom `ui-namespace` on the ViewList to use a different viewdef (e.g., one without the delete button):

```html
<!-- ViewListItem.readonly viewdef (no delete button) -->
<template>
  <div ui-view="item" ui-namespace="list-item"></div>
</template>
```

**ViewList frontend behavior:**

The frontend ViewList widget:
- Has an **exemplar element** that gets cloned for each ViewListItem (default: `<div>`)
- Each cloned element is rendered as a View using standard `render()` (gets `ui-viewdef` attribute, supports hot-reload)
- Maintains a parallel array of View elements for the ViewListItems
- When ViewListItems are added/removed, updates the DOM accordingly

**Exemplar namespace inheritance:**

ViewList exemplars follow the standard namespace inheritance rules for views:
- The exemplar's variable inherits `namespace` from the ViewList's variable (unless the exemplar specifies `ui-namespace`)
- The exemplar's variable inherits `fallbackNamespace` from the ViewList's variable

This allows a ViewList with `ui-namespace="COMPACT"` to render all its items using `TYPE.COMPACT` viewdefs without requiring each exemplar to specify the namespace.

**Example: Custom namespace for domain objects:**

```html
<div ui-view="customers?wrapper=lua.ViewList" ui-namespace="customer-item"></div>
```

With this setup and standard `lua.ViewList.list-item.html` viewdef:
1. ViewList variable gets `namespace: "customer-item"` and `fallbackNamespace: "list-item"` (from backend wrapper)
2. When rendering ViewList: tries `lua.ViewList.customer-item` → falls back to `lua.ViewList.list-item`
3. ViewListItem variable inherits both properties
4. When rendering ViewListItem: tries `lua.ViewListItem.customer-item` → falls back to `lua.ViewListItem.list-item`
5. The inner `ui-view="item"` for Customer inherits both properties
6. When rendering Customer: tries `Customer.customer-item` (exists) → uses it

This allows custom domain object viewdefs (e.g., `Customer.customer-item.html`) without needing to define `lua.ViewList.customer-item.html` or `lua.ViewListItem.customer-item.html` - those fall back to their `list-item` viewdefs.

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
