# UI-Engine Usage Guide

## Contents

- [Why ui-engine](#why-ui-engine)
- [Getting Started](#getting-started)
  - [Project Structure](#project-structure)
  - [Running the Server](#running-the-server)
- [Core Concepts](#core-concepts)
  - [Where to Put Logic](#where-to-put-logic)
  - [Domain vs Presenter](#domain-vs-presenter)
  - [Variable Creation Flow](#variable-creation-flow)
  - [Path Resolution](#path-resolution)
- [Lua Backend](#lua-backend)
  - [Defining Types](#defining-types)
  - [Hot-Reloading](#hot-reloading-lua-code)
- [Template Reference](#template-reference)
  - [Value Bindings](#value-bindings)
  - [Actions and Events](#actions-and-events)
  - [Path Properties](#path-properties)
  - [Views and Namespaces](#views-and-namespaces)
  - [Lists with ViewList](#lists-with-viewlist)
  - [Variable Wrappers](#variable-wrappers)
- [Complete Example](#complete-example-contact-manager)
- [CLI Reference](#cli-reference)

---

## Why ui-engine

This is not your grandfather's web framework.

Traditional frameworks treat UI as pages, routes, and components. ui-engine treats UI as **objects presenting themselves**. A `Contact` object knows how to render as a detail view, a list item, an editor, or a dropdown option—each context gets its own presentation, but it's still the same object.

```
Contact object → presents as:
  ├── Contact.DEFAULT.html      (full detail view)
  ├── Contact.list-item.html    (compact row in a list)
  ├── Contact.editor.html       (editable form)
  └── Contact.option.html       (dropdown option)
```

**The paradigm shift:**
- Objects define their own views, not pages
- Same object, different contexts, different presentations
- No routing—just objects rendering where they're referenced
- No state management—objects ARE the state

**What this enables:**
- **No frontend JavaScript** - Declarative HTML templates with `ui-*` attributes + Lua backend
- **Automatic change detection** - modify objects directly, UI updates
- **Hot-reloading** - edit Lua or HTML, see changes instantly
- **Backend is source of truth** - frontend just renders what backend provides

---

## Getting Started

### Project Structure

When running with `--dir`, ui-engine expects this layout:

```
my-project/
├── html/                              # Static web assets (served at root URL)
│   ├── index.html                     # Main HTML page
│   ├── main-*.js                      # Frontend JavaScript
│   └── worker-*.js                    # Web worker scripts
├── lua/                               # Lua backend scripts
│   └── main.lua                       # Entry point - loaded per session
└── viewdefs/                          # HTML view templates
    ├── MyType.DEFAULT.html            # Default view for MyType
    ├── MyType.list-item.html          # Named namespace view
    └── lua.ViewListItem.*.html        # ViewList item templates
```

| Directory   | Purpose                                            |
|-------------|----------------------------------------------------|
| `html/`     | Static files served at the web root                |
| `lua/`      | Lua scripts; `main.lua` is loaded for each session |
| `viewdefs/` | HTML templates for rendering types                 |

**Viewdef naming:** `TypeName.Namespace.html`
- `TypeName` — the Lua type name (from `session:prototype()`)
- `Namespace` — `DEFAULT` or a custom name like `list-item`

### Running the Server

```bash
# Development with hot-reloading
ui-engine --port 8000 --dir my-project/ --hotload

# Production
ui-engine --port 8080 --dir my-project/
```

This serves:
- `http://localhost:8000/` → `my-project/html/index.html`
- Lua scripts from `my-project/lua/`
- Viewdefs from `my-project/viewdefs/`

---

## Core Concepts

### Where to Put Logic

| Location       | Use For                                     | Trade-offs                      |
|----------------|---------------------------------------------|---------------------------------|
| **Lua**        | All behavior whenever possible              | Simple, responsive, portable    |
| **JavaScript** | Browser APIs, DOM tricks beyond ui-engine   | Last resort, harder to maintain |

**Prefer Lua.** Lua methods execute instantly when users click buttons or type.

```lua
-- Lua handles form validation instantly
function ContactApp:save()
    if self.name == "" then
        self.error = "Name required"
        return
    end
    table.insert(self.contacts, Contact:new({name = self.name, email = self.email}))
    self:clearForm()
end
```

**Use JavaScript only for:** browser capabilities not in ui-engine, custom DOM manipulation, browser APIs (clipboard, notifications, downloads).

**JavaScript is available via:**
- `<script>` elements in viewdefs — static "library" code loaded once
- `ui-code` attribute — dynamic injection as-needed

### Domain vs Presenter

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

### Path Resolution

Variable values are **object references** (`{"obj": 1}`), not actual data. All path-based bindings MUST create child variables - the backend resolves paths, not the frontend.

**Why object references?**
- Enables object identity tracking across the UI
- Same object can appear in multiple places
- Supports circular references without infinite serialization
- Change detection compares object identity, not deep equality

---

## Lua Backend

### Defining Types

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

### Hot-Reloading Lua Code

With `--hotload` enabled, Lua files reload automatically when saved. Use `session:prototype()` and `session:create()` for automatic state preservation.

**Key behavior:**
- Only files already loaded by the session are reloaded (new files are ignored until `require`d)
- `session.reloading` is `true` during reload, `false` otherwise

**Basic Pattern:**

```lua
-- 1. Declare prototypes (assign for LSP support)
Person = session:prototype("Person", {
    name = "",
    email = "",
    avatar = EMPTY,  -- EMPTY: starts nil, but tracked for mutation
})

-- Prototype variables are assigned separately (not in init)
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

Use `EMPTY` to declare optional fields that start nil but are tracked for mutation.

**Adding Fields:** Add to the prototype. Existing instances inherit via metatable.

**Removing Fields:** Remove from prototype. Framework nils out the field on all instances automatically.

**Renaming/Migrating Fields:** Use `mutate()` method (called automatically after reload):

```lua
function Person:mutate()
    if self.fullName then
        self.name = self.name or self.fullName
        self.fullName = nil
    end
end
```

**Rules:**
1. Always use `session:prototype()` — not `X = X or {}`
2. Override `:new()` only when needed — default calls `session:create()` automatically
3. Guard app creation — `if not session:getApp() then`
4. `mutate()` must be idempotent — safe to call multiple times

See `HOT-LOADING.md` for design details.

### Hot-Reloading Viewdefs

Viewdefs also reload automatically during development:
- Edit an HTML viewdef file → UI updates immediately
- No page refresh required
- State is preserved (variables, form inputs, scroll positions)

---

## Template Reference

### Value Bindings

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

### Actions and Events

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

### Path Properties

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

| Property              | Description                                          |
|-----------------------|------------------------------------------------------|
| `keypress`            | Send updates on every keystroke (not just blur)      |
| `scrollOnOutput`      | Auto-scroll element to bottom when value updates     |
| `wrapper=TypeName`    | Use a wrapper for value transformation               |
| `itemWrapper=TypeName`| Wrap each list item with a presenter                 |
| `create=TypeName`     | Create instance as variable value                    |
| `access=r\|w\|rw\|action` | Control read/write behavior                      |
| `replace`             | For `ui-html`, replace element instead of innerHTML  |

**Default ui-value access by element type:**
- Native inputs (`input`, `textarea`, `select`): `rw`
- Interactive Shoelace (`sl-input`, `sl-textarea`, `sl-select`, `sl-checkbox`, `sl-switch`, etc.): `rw`
- Read-only Shoelace (`sl-progress-bar`, `sl-progress-ring`, `sl-qr-code`): `r`
- Non-interactive elements (`div`, `span`, etc.): `r`

**Read/Write Method Paths:**

Methods can act as read/write properties by ending the path in `()` with `access=rw`:

```html
<input ui-value="value()?access=rw">
```

```lua
function MyPresenter:value(...)
    if select('#', ...) > 0 then
        self._value = select(1, ...)  -- write
    end
    return self._value  -- read
end
```

### Views and Namespaces

**Multi-Element Templates:**

View templates can contain multiple top-level elements. The first element receives the variable's ID; additional elements get auto-vended IDs:

```html
<!-- lua.ViewListItem.my-options.html -->
<template>
  <sl-option ui-attr-value="index">
    <span ui-value="item.name"></span>
  </sl-option>
</template>
```

**Viewdef Naming:** `Type.Namespace.html`
- `Contact.DEFAULT.html` - full detail view
- `Contact.list-item.html` - compact list row

**Namespace Resolution (3-tier):**
1. Variable's `namespace` property → `Type.{namespace}`
2. Variable's `fallbackNamespace` property → `Type.{fallbackNamespace}`
3. Default → `Type.DEFAULT`

```html
<!-- Explicit namespace -->
<div ui-view="contact" ui-namespace="COMPACT"/>

<!-- ViewList sets fallbackNamespace="list-item" automatically -->
<div ui-viewlist="contacts"/>
```

### Lists with ViewList

**Recommended pattern**: Backend holds domain objects, ViewList creates presenter layer.

```html
<!-- Basic list -->
<div ui-viewlist="contacts"/>

<!-- With custom item presenter -->
<div ui-viewlist="contacts?itemWrapper=ContactPresenter" ui-namespace="list-item"/>
```

**How ViewList Works:**

```
Backend domain data:
  contacts: [{obj: 101}, {obj: 102}]  <- Contact refs

ViewList creates ViewItem for each:
  ViewItem:
    ├── baseItem: {obj: 101}     <- domain object ref
    ├── item: {obj: P1}          <- wrapped presenter (if itemWrapper set)
    ├── list: ViewList           <- back-reference
    └── index: 0                 <- position
```

**ViewList for `<sl-select>` Options:**

```html
<sl-select ui-value="selectedContactId" label="Contact">
  <span ui-view="contacts()?wrapper=lua.ViewList" ui-namespace="contact-option"></span>
</sl-select>
```

```html
<!-- lua.ViewListItem.contact-option.html -->
<template>
  <sl-option ui-attr-value="index">
    <span ui-value="item.fullName()"></span>
  </sl-option>
</template>
```

Key points:
- Use a `<span>` as the container (it gets replaced by the options)
- The viewdef is for `lua.ViewListItem`, not your domain type
- `index` is 0-based position; `item` is the wrapped object; `baseItem` is unwrapped

### Variable Wrappers

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

---

## Complete Example: Contact Manager

### Backend (Lua)

```lua
-- Domain object
Contact = session:prototype("Contact", {
  firstName = "",
  lastName = "",
  email = "",
  phone = ""
})

function Contact:fullName()
  return self.firstName .. " " .. self.lastName
end

-- App presenter
ContactApp = session:prototype("ContactApp", {
  contacts = {},
  selectedIndex = EMPTY,
  searchQuery = ""
})

function ContactApp:addContact()
  table.insert(self.contacts, Contact:new({
    firstName = "New",
    lastName = "Contact"
  }))
end

function ContactApp:selectedContact()
  if self.selectedIndex then
    return self.contacts[self.selectedIndex]
  end
end

-- Initialize
if not session:getApp() then
  session:createAppVariable(ContactApp:new())
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

---

## CLI Reference

```
Usage: ui-engine [command] [options]

Commands:
  serve      Start the UI server (default)
  bundle     Create binary with custom site bundled
  extract    Extract bundled site to filesystem
  ls         List files in bundled site
  cat        Display contents of a bundled file
  cp         Copy files from bundled site

Server Options:
  --host              Listen address (default: 0.0.0.0)
  --port              Listen port (default: 8080)
  --dir               Serve from directory instead of embedded site
  --hotload           Enable hot-reloading of Lua and viewdef files
  --lua               Enable Lua backend (default: true)
  --lua-path          Lua scripts directory
  --socket            Backend API socket path
  --session-timeout   Session expiration (default: 24h, 0=never)
  --log-level         Log level: debug, info, warn, error

Examples:
  ui-engine --port 8080 --dir my-site/ --hotload
  ui-engine bundle site/ -o my-app
  ui-engine extract extracted/
```

**Protocol Commands** (for testing/debugging):
```
  create     Create a new variable
  destroy    Destroy a variable
  update     Update a variable
  watch      Watch a variable
  get        Get variable values
  poll       Poll for pending responses
```
