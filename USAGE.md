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
local Contact = {type = "Contact"}
Contact.__index = Contact

function Contact:new(tbl)
  tbl = tbl or {}
  setmetatable(tbl, self)
  return tbl
end

function Contact:fullName()
  return self.firstName .. " " .. self.lastName
end
```

- `type` field in metatable enables viewdef resolution
- `new(tbl)` pattern for instantiation
- Methods callable via `ui-action` paths

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
```

**ui-code bindings:**
- Execute JavaScript when the bound value changes
- Code has access to: `element`, `value`, `variable`, `store`
- Use for advanced DOM manipulation the declarative bindings can't handle

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

The wrapper:
- Receives the **variable** (not just value), enabling it to watch for changes
- Computes the outgoing JSON value sent to frontend
- Can create/manage additional objects (like presenters)

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

**Lua code reloading:**
- Lua's evaluation model allows code redefinition
- Extensions can provide hot-reload hooks (e.g., ui-mcp)

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
