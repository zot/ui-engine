# ui-engine Demos

## Running

From the project root:

```bash
make demo
./build/ui-engine-demo --port 8000 --dir demo
```

Then open http://localhost:8000 in your browser.

*If you want verbose logging, you can use `-v` up to `-vvvv`.*

## Available Demos

### Contact Manager (default)

Full-featured demo showing domain/presenter separation, ViewList, search, forms.

**Entry point:** `lua/main.lua`

### Simple Adder

Minimal demo showing basic bindings and computed values.

**To use:** Replace `main.lua` with `adder.lua`:

```bash
cp demo/lua/adder.lua demo/lua/main.lua
```

Two inputs that add together - demonstrates:
- `ui-value` two-way binding
- Computed method binding: `ui-value="compute()"`

```html
<input ui-value="value1">
+
<input ui-value="value2">
=
<span ui-value="compute()"></span>
```

## What It Demonstrates

### Domain vs Presenter Separation

- **Contact** (`lua/main.lua`) - Domain object with `firstName`, `lastName`, `email`, `phone`, `notes`
- **ContactPresenter** - Wraps Contact for UI actions (`edit()`, `delete()`)
- **ContactApp** - App presenter managing view state and contact list

### Declarative Bindings

From `viewdefs/ContactApp.DEFAULT.html`:

```html
<!-- Two-way binding with live search -->
<sl-input ui-value="searchQuery?keypress" ui-event-keypress-enter="selectFirstContact()">

<!-- Call backend method -->
<sl-button ui-action="addContact()">Add Contact</sl-button>

<!-- ViewList with item wrapper -->
<div ui-view="contacts()?wrapper=lua.ViewList&itemWrapper=ContactPresenter"></div>

<!-- Conditional visibility -->
<div ui-attr-hidden="isEditView">...</div>
```

### ViewList Pattern

The contact list uses ViewList to automatically wrap domain objects:

```html
<div ui-view="contacts()?wrapper=lua.ViewList&itemWrapper=ContactPresenter"></div>
```

- `contacts()` returns filtered domain objects
- `wrapper=lua.ViewList` manages the list
- `itemWrapper=ContactPresenter` wraps each contact for UI actions

### Hot-Reloading

Edit any viewdef file while the app is running - changes appear immediately without refresh.

## File Structure

```
demo/
├── lua/
│   ├── main.lua          # Entry point (copy or link contact.lua or adder.lua here)
│   ├── contact.lua       # Contact Manager backend
│   └── adder.lua         # Simple Adder backend
├── viewdefs/
│   ├── ContactApp.DEFAULT.html        # Contact Manager main view
│   ├── ContactPresenter.list-item.html # Contact row in list
│   └── Adder.DEFAULT.html             # Simple Adder view
└── html/
    └── index.html        # Entry point (loads ui-engine frontend)
```

## Key Patterns

1. **Backend creates only variable 1** - `session:createAppVariable(app)`
2. **Viewdefs create all other variables** - via `ui-view`, `ui-value` paths
3. **Direct mutation** - just modify objects, UI updates automatically
4. **No frontend JS** - all logic in Lua, all UI in HTML templates
