# UI-Engine Usage Guide

## Why ui-engine

Build reactive web UIs using only backend code and declarative HTML templates.

- **No frontend JavaScript required** - just HTML templates with `ui-*` attributes + Lua backend
- **Declarative bindings** - `ui-value`, `ui-action`, `ui-view`, etc.
- **Automatic change detection** via `change-tracker`:
  - Modify objects directly - UI updates automatically
  - No observer pattern required - no boilerplate
  - Works with Go and Lua; portable to any language with reflection (Python, Java, JS)
- **Hot-reloading viewdefs** - edit HTML, see changes instantly (state preserved)
- **Backend is source of truth** - frontend just renders what backend provides
- **Current focus**: embedded Lua (supports ui-mcp project)

## Philosophy

### Where to Put Logic

Behavior can exist in 2 places:

| Location       | Use For                                           | Trade-offs                    |
|----------------|---------------------------------------------------|-------------------------------|
| **Lua**        | All behavior whenever possible                    | Simple, responsive, portable  |
| **JavaScript** | Browser APIs, DOM tricks beyond ui-engine         | Last resort, harder to maintain |

**Prefer Lua.** Lua methods execute instantly when users click buttons or type.

```lua
-- GOOD: Lua handles form validation instantly
function ContactApp:save()
    if self.name == "" then
        self.error = "Name required"
        return
    end
    table.insert(self.contacts, Contact:new({name = self.name, email = self.email}))
    self:clearForm()
end

-- GOOD: Lua handles UI state changes instantly
function ContactApp:toggleSection()
    self.sectionExpanded = not self.sectionExpanded
end
```

**Use JavaScript only for:**
- Browser capabilities not in ui-engine (e.g., if `scrollOnOutput` didn't exist)
- Custom DOM manipulation
- Browser APIs (clipboard, notifications, downloads)

**JavaScript is available via:**
- `<script>` elements in viewdefs — static "library" code loaded once
- `ui-code` attribute — dynamic injection as-needed (see ui-code bindings below)

### Domain vs Presenter Separation

- **Domain objects** hold app data and core behavior (e.g., `Contact` with firstName, email)
- **Presenter objects** wrap domain objects and add UI state/behavior (e.g., `ContactPresenter` adds `delete()`, `isEditing`)
- Domain objects should NOT have UI-specific methods

### Variable Creation Flow

**Critical insight: The frontend creates most variables, not the backend.**

1. Backend creates only **variable 1** (app presenter with domain data)
2. Backend sends viewdefs for presenter types
3. Frontend renders viewdefs containing paths like `ui-view="selectedContact"`
4. These paths create variables that "reach into" backend objects
5. Path resolution may call methods, returning presenters

*"The backend creating variables when instructed by the frontend which, in turn, gets those from the viewdefs it parses from the backend."*

### Path Resolution: Server-Side Only

Variable values are **object references** (`{"obj": 1}`), not actual data. All path-based bindings MUST create child variables - the backend resolves paths, not the frontend.

**Why object references?**
- Enables object identity tracking across the UI
- Same object can appear in multiple places
- Supports circular references without infinite serialization
- Change detection compares object identity, not deep equality

## Defining Types (Lua)

```lua
Contact = session:prototype("Contact", {
  firstName = "",
  lastName = "",
  email = "",
})

function Contact:fullName()
  return self.firstName .. " " .. self.lastName
end
```

- `session:prototype()` handles `type`, `__index`, and default `:new()` automatically
- Methods callable via `ui-action` paths
- Fields prefixed with `_` (e.g., `_cache`) are private — not serialized to frontend

## Value Bindings

```html
<!-- Two-way binding for inputs, one-way for display -->
<sl-input ui-value="firstName"/>
<span ui-value="fullName()"/>

<!-- Attribute bindings -->
<button ui-attr-disabled="isLocked">Submit</button>

<!-- Class bindings -->
<div ui-class-active="isSelected">Item</div>

<!-- Style bindings -->
<div ui-style-color="themeColor">Styled</div>

<!-- Code bindings - execute JS when value changes -->
<div ui-code="formatHandler"/>

<!-- HTML bindings - inject HTML content -->
<div ui-html="description"/>
<div ui-html="renderedMarkdown?replace"/>
```

**ui-code bindings:**
- Execute JavaScript when the bound value changes
- Code has access to: `element`, `value`, `variable`, `store`
- Use for advanced DOM manipulation the declarative bindings can't handle

**ui-html bindings:**
- Inject HTML content from a variable into an element
- Standard mode: sets `innerHTML` (element preserved)
- Replace mode (`?replace`): replaces the element itself with the HTML content
  - First element in HTML gets the original element's ID
  - Multiple elements (fragments) are supported - additional elements get auto-vended IDs
  - Empty content creates a hidden placeholder
- Read-only by default (`access=r`)

## Actions and Events

```html
<!-- Call backend method on click -->
<sl-button ui-action="save()">Save</sl-button>

<!-- Call with argument -->
<sl-button ui-action="delete(contact)">Delete</sl-button>

<!-- Set variable on click -->
<div ui-event-click="selectedIndex" ui-event-click-value="0">First</div>

<!-- Keypress events -->
<sl-input ui-event-keypress-enter="submit()"/>

<!-- Modifier combinations -->
<sl-input ui-event-keypress-ctrl-s="save()"/>
<sl-input ui-event-keypress-ctrl-shift-z="redo()"/>
```

Available modifiers: `ctrl`, `shift`, `alt`, `meta`

## Path Properties

Paths can include properties: `path?property=value&other=value`

```html
<!-- Live updates on every keystroke (default is blur) -->
<sl-input ui-value="search?keypress"/>

<!-- Auto-scroll to bottom when value updates -->
<pre ui-value="log?scrollOnOutput"></pre>

<!-- Use a wrapper for value transformation -->
<div ui-view="contact?wrapper=ContactPresenter"/>

<!-- Wrap list items with presenter -->
<div ui-viewlist="contacts?itemWrapper=ContactPresenter"/>

<!-- Create instance as variable value -->
<div ui-view="newContact?create=Contact"/>

<!-- Control read/write behavior -->
<span ui-value="total?access=r"/>
```

**Common properties:**
- `keypress` - send updates on every keystroke (not just blur)
- `scrollOnOutput` - auto-scroll element to bottom when value updates
- `wrapper=TypeName` - use a wrapper for value transformation
- `itemWrapper=TypeName` - wrap each list item with a presenter
- `create=TypeName` - create instance as variable value
- `access=r|w|rw|action` - control read/write behavior
- `replace` - for `ui-html`, replace the element with the HTML content instead of setting innerHTML

**Default ui-value access by element type:**
- Native inputs (`input`, `textarea`, `select`): `rw`
- Interactive Shoelace (`sl-input`, `sl-textarea`, `sl-select`, `sl-checkbox`, `sl-radio`, `sl-radio-group`, `sl-radio-button`, `sl-switch`, `sl-range`, `sl-color-picker`, `sl-rating`): `rw`
- Read-only Shoelace (`sl-progress-bar`, `sl-progress-ring`, `sl-qr-code`, `sl-option`, `sl-copy-button`): `r`
- Non-interactive elements (`div`, `span`, etc.): `r`

### Read/Write Method Paths

Methods can act as read/write properties by ending the path in `()` with `access=rw`:

```html
<input ui-value="value()?access=rw">
```

On read, the method is called with no arguments. On write, the value is passed as an argument. In Lua, use varargs:

```lua
function MyPresenter:value(...)
    if select('#', ...) > 0 then
        self._value = select(1, ...)  -- write
    end
    return self._value  -- read
end
```

## Views and Namespaces

### Viewdef Naming

Viewdefs are named `Type.Namespace.html`:
- `Contact.DEFAULT.html` - full detail view
- `Contact.list-item.html` - compact list row
- `Contact.OPTION.html` - for `<sl-select>` options

### Namespace Resolution (3-tier)

1. Variable's `namespace` property → `Type.{namespace}`
2. Variable's `fallbackNamespace` property → `Type.{fallbackNamespace}`
3. Default → `Type.DEFAULT`

```html
<!-- Explicit namespace -->
<div ui-view="contact" ui-namespace="COMPACT"/>

<!-- ViewList sets fallbackNamespace="list-item" automatically -->
<div ui-viewlist="contacts"/>
```

## Lists with ViewList

**Recommended pattern**: Backend holds domain objects, ViewList creates presenter layer.

```html
<!-- Basic list -->
<div ui-viewlist="contacts"/>

<!-- With custom item presenter -->
<div ui-viewlist="contacts?itemWrapper=ContactPresenter" ui-namespace="list-item"/>
```

### How ViewList Works

```
Backend domain data:
  contacts: [{obj: 101}, {obj: 102}]  <- Contact refs

ViewList creates ViewItem for each:
  ViewItem:
    ├── baseItem: {obj: 101}     <- domain object ref
    ├── item: {obj: P1}          <- wrapped presenter (if itemWrapper set)
    ├── list: ViewList           <- back-reference
    └── index: 0                 <- position

ViewItem viewdef uses ui-view="item" to display the presenter
```

### Why This Approach

1. **Separation of concerns** - backend handles domain logic, frontend handles presentation
2. **Flexibility** - same domain list can have different presenters in different views
3. **Simpler backend** - no presenter management code in Lua
4. **Automatic sync** - ViewList keeps presenters in sync with domain array changes

## Variable Wrappers

The `wrapper` property enables value transformation at the backend.

```html
<!-- Wrap single object in presenter -->
<div ui-view="contact?wrapper=ContactPresenter"/>

<!-- Custom computed display -->
<span ui-value="items?wrapper=CountDisplay"/>  <!-- shows "3 items" -->
```

**Creating a Wrapper:**

```lua
MyWrapper = session:prototype("MyWrapper", {
  variable = EMPTY,  -- the Variable object
  value = EMPTY,     -- convenience: variable's current value
})

function MyWrapper:new(variable)
  -- Check for existing wrapper to preserve state across value changes
  local existing = variable:getWrapper()
  if existing then
    existing.value = variable:getValue()
    return existing
  end

  local wrapper = session:create(MyWrapper)
  wrapper.variable = variable
  wrapper.value = variable:getValue()
  return wrapper
end
```

The wrapper:
- Receives the **variable** (not just value), enabling it to watch for changes
- Should check `variable:getWrapper()` to reuse existing wrapper and preserve state
- Child variable paths navigate from the wrapper object

ViewList is a built-in wrapper that handles arrays automatically.

## Development Workflow

### Hot-Reloading Viewdefs

Viewdefs reload automatically during development:
- Edit an HTML viewdef file → UI updates immediately
- No page refresh required
- State is preserved (variables, form inputs, scroll positions)

**Rapid iteration:**
1. Open app in browser
2. Edit viewdef HTML in your editor
3. Save → see changes instantly

### Hot-Reloading Lua Code

With `--hotload` enabled, Lua files reload automatically when saved. Use `session:prototype()` and `session:create()` for automatic state preservation.

**Key behavior:**
- Only files already loaded by the session are reloaded (new files are ignored until `require`d)
- `session.reloading` is `true` during reload, `false` otherwise — use this to detect hot-reloads in your code

**Basic Pattern:**

```lua
-- 1. Declare prototypes (assign for LSP support)
-- Prototypes get a default :new(instance) method automatically
-- init declares instance fields — only these are tracked for mutation
Person = session:prototype("Person", {
    name = "",
    email = "",
    avatar = EMPTY,  -- EMPTY: starts nil, but tracked for mutation
})

-- Prototype variables are assigned separately (not in init)
-- These are shared across instances, not per-instance defaults
Person.nextId = Person.nextId or 0

-- 2. Override :new() when you need custom initialization
function Person:new(instance)
    instance = session:create(Person, instance)
    instance.id = Person.nextId
    Person.nextId = Person.nextId + 1
    return instance
end

-- 3. Guard app creation
if not session:getApp() then
    session:createAppVariable(App:new())
end
```

Use `EMPTY` to declare optional fields that start nil but are tracked for mutation. When you remove a field from init, it's nil'd out on all instances.

Save the file → instances get new methods immediately.

**Adding Fields:**

Add to the prototype. Existing instances inherit via metatable:

```lua
Person = session:prototype("Person", {
    name = "",
    email = "",
    avatar = EMPTY,
    phone = "",  -- NEW: inherited automatically
})
```

If instances need computed values, add a `mutate` method (called automatically after reload):

```lua
function Person:mutate()
    self.phone = self.phone or "unknown"
end
```

**Removing Fields:**

Remove from prototype. Framework nils out the field on all instances automatically.

**Renaming/Migrating Fields:**

```lua
function Person:mutate()
    if self.fullName then
        self.name = self.name or self.fullName
        self.fullName = nil
    end
end
```

**Rules:**
1. **Always use `session:prototype()`** — not `X = X or {}`
2. **Override `:new()` only when needed** — default calls `session:create()` automatically
3. **Guard app creation** — `if not session:getApp() then`
4. **`mutate()` must be idempotent** — safe to call multiple times
5. **Prototype order matters** — declare dependencies first

**What hot-loading enables:**
- Add/modify methods — immediately available on existing instances
- Fix bugs — corrections take effect without restart
- Schema migrations — automatic via `mutate()`

**What hot-loading cannot do:**
- Change metatable identity — instances keep same metatable reference (which is the point)

See `HOT-LOADING.md` for design details and implementation notes.

## Complete Example: Contact Manager

### Backend (Lua)

```lua
-- Domain object
local Contact = {type = "Contact"}
Contact.__index = Contact

function Contact:new(tbl)
  return setmetatable(tbl or {}, self)
end

function Contact:fullName()
  return self.firstName .. " " .. self.lastName
end

-- App presenter
local ContactApp = {type = "ContactApp"}
ContactApp.__index = ContactApp

function ContactApp:new()
  return setmetatable({
    contacts = {},
    selectedIndex = nil,
    searchQuery = ""
  }, self)
end

function ContactApp:addContact()
  table.insert(self.contacts, Contact:new({
    firstName = "New",
    lastName = "Contact",
    email = "",
    phone = ""
  }))
end

function ContactApp:selectedContact()
  if self.selectedIndex then
    return self.contacts[self.selectedIndex]
  end
end
```

### Viewdefs

**ContactApp.DEFAULT.html:**
```html
<div class="contact-manager">
  <sl-input ui-value="searchQuery?keypress" placeholder="Search..."/>
  <sl-button ui-action="addContact()">Add Contact</sl-button>

  <!-- List with presenter wrapping -->
  <div ui-viewlist="contacts?itemWrapper=ContactPresenter" ui-namespace="list-item"/>

  <!-- Selected contact detail -->
  <div ui-view="selectedContact()"/>
</div>
```

**ContactPresenter.list-item.html:**
```html
<div class="contact-row" ui-action="select()">
  <span ui-value="item.fullName()"/>
  <span ui-value="item.email"/>
  <sl-icon-button name="trash" ui-action="delete()"/>
</div>
```

**Contact.DEFAULT.html:**
```html
<div class="contact-detail">
  <h2 ui-value="fullName()"/>
  <sl-input ui-value="firstName" label="First Name"/>
  <sl-input ui-value="lastName" label="Last Name"/>
  <sl-input ui-value="email" label="Email"/>
  <sl-input ui-value="phone" label="Phone"/>
</div>
```

## Key Principles

1. **Backend stores domain objects** - simple data, no UI concerns
2. **Frontend creates variables via viewdefs** - paths in viewdefs cause variable creation
3. **ViewList wraps domain arrays** - automatically creates presenter objects
4. **Presenters add UI state/behavior** - `isEditing`, `delete()`, etc.
5. **Path properties configure behavior** - `?itemWrapper=`, `?keypress`, etc.
6. **No frontend JavaScript** - just HTML templates with `ui-*` attributes
