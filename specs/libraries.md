# Libraries

## Backend Library

The backend library makes integrating with the UI server easy. Provided for **Go** and **Lua**.

**Connection:**
- Connect to UI server with a root value for variable `1`
- Invokes hook upon connection close
- Root value must bind to `currentPage()`
- If the app is an SPA, the frontend will bind to `historyIndex` and `url` on the root

**Path navigation:**
- Handles path navigation with reflection

**Change detection** (provided by `change-tracker` package - `github.com/zot/change-tracker`):
- Variables hold references to backend objects, not copies of data
- Backend code modifies objects directly - no manual `update()` calls needed
- After processing a batch of messages, the framework:
  1. Computes current values for all watched variables
  2. Detects which values have changed since last computation
  3. Automatically sends update messages for changed variables
- Does not require support for the observer pattern, allowing any backend object to support variables
- Refreshes happen automatically after receipt of client messages
- Background-triggered changes are throttled
- Provides a thread-safe mechanism for interacting with refresh logic
- **Implementation note**: The `change-tracker` package handles variable tracking, change detection, and update dispatch. Design documents should reference this package rather than re-specifying the algorithm.

**Object registry (Go):**
- Maps backend objects to variable IDs using weak references (Go 1.25+ `weak` package)
- Objects have identity independent of where they appear - the same object in multiple locations serializes to the same `{"obj": id}`
- When a path is watched, the object at that path is registered with its variable ID
- During serialization, objects found in the registry emit `{"obj": id}` instead of inline values
- Weak references ensure objects can be garbage collected when no longer referenced by application code
- The registry is automatically cleaned up as objects are collected
- **Frictionless**: domain objects require no modification - no interfaces, no embedded IDs

## Lua Session API

The embedded Lua runtime provides a `session` global for variable management. This is available when `main.lua` executes for each new frontend session.

**Automatic Change Detection** (via `change-tracker` package):

Variables hold references to Lua objects. The framework automatically detects and propagates changes:
- Backend methods modify objects directly (e.g., `self.title = "New Title"`)
- After processing each batch of messages, the framework computes current values for watched variables
- Changed values are automatically sent to the frontend
- No manual `update()` calls are needed for value changes
- See "Change detection" in Backend Library section for implementation details

**Session object:**
```lua
-- Create the app variable (variable 1) - typically done in main.lua
-- The variable holds a reference to the app object
session:createAppVariable(app)

-- Get the app object (the actual Lua table, not a wrapper)
local app = session:getApp()

-- Create a child variable pointing to an object
session:createVariable(parentId, object)

-- Destroy a variable
session:destroyVariable(id)

-- Log a message (delegates to Config.Log)
session:log(level, message)
```

**Built-in property watchers:**

The Lua runtime automatically watches the `lua` property on variable 1. When updated:
- If value ends with `.lua`, loads the file from `<site>/lua/<filename>`
- Otherwise, executes the value as inline Lua code

**Lua Type Conventions:**

Lua types follow a convention that enables frictionless development:

1. **Type field in metatable**: Each type defines a `type` field in its metatable table
2. **`new(tbl)` constructor**: Use `Type:new(tbl)` where `tbl` is an optional table for the instance
3. **Auto-extraction**: `createAppVariable` and `createVariable` automatically extract the type from the object's `type` field

```lua
-- Define a type with metatable pattern
local Item = {type = "Item"}  -- type field in metatable
Item.__index = Item

function Item:new(tbl)
  tbl = tbl or {}
  setmetatable(tbl, self)  -- tbl inherits type from metatable
  tbl.name = tbl.name or ""
  return tbl
end

-- Define the app type
local App = {type = "App"}
App.__index = App

function App:new(tbl)
  tbl = tbl or {}
  setmetatable(tbl, self)
  tbl.title = tbl.title or "My Application"
  tbl.items = tbl.items or {}  -- array of Item objects
  return tbl
end

-- Methods callable via ui-action paths
-- Just modify self directly - changes are auto-detected
function App:addItem(name)
  local item = Item:new({name = name})
  session:createVariable(self, item)  -- creates child variable for item
  table.insert(self.items, item)      -- add to items array
  -- No update() call needed - framework detects the change
end

function App:deleteItem(index)
  local item = self.items[index]
  if item then
    session:destroyVariable(item)     -- destroy the child variable
    table.remove(self.items, index)   -- remove from array
    -- No update() call needed - framework detects the change
  end
end

-- Create app and register as app variable
local app = App:new({title = "My Application"})
session:createAppVariable(app)
```

**Key points:**
- `type` is a **variable property** (metadata), not part of the JSON value
- The frontend uses the type property to resolve viewdefs: `{type}.{namespace}.html`
- Namespace defaults to `DEFAULT` and can be overridden with `ui-namespace` attribute
- Internal/private fields should be prefixed with `_` (e.g., `_contactData`) - these are not serialized
- **No manual update() calls** - just modify objects directly and changes are auto-detected

## Lua Wrapper Types

Wrappers stand in for variable values when child variables navigate paths. The wrapper object itself is registered and becomes the navigation value. Lua wrappers follow a convention similar to regular types:

1. **`new(variable)` constructor**: Wrapper receives the change-tracker Variable object
2. **`variable` property**: Store the variable for later access
3. **`value` property**: Store the variable's Value for convenience

```lua
-- Define a wrapper type
local MyWrapper = {type = "MyWrapper"}
MyWrapper.__index = MyWrapper

function MyWrapper:new(variable)
  local wrapper = {
    variable = variable,        -- store the Variable object
    value = variable:getValue() -- store Value for convenience
  }
  setmetatable(wrapper, self)
  return wrapper
end

-- The wrapper's fields are accessible via child variable paths
-- Use in viewdef: ui-view="item?wrapper=MyWrapper"
```

The wrapper's `variable` property provides access to:
- `variable:getID()` - variable ID
- `variable:getValue()` - current Value (same as stored `value`)
- `variable:getProperty(name)` - variable properties (e.g., "item", "path")
- `variable:getWrapper()` - existing wrapper if any (for reuse)

When a variable has a wrapper, child variable paths navigate from the wrapper object instead of the raw value.

**Wrapper reuse pattern:**

`CreateWrapper` is called whenever the variable's value changes. Stateful wrappers (like ViewList) should check for an existing wrapper and reuse it to preserve internal state:

```lua
function MyListWrapper:new(variable)
  -- Check for existing wrapper to preserve state
  local existing = variable:getWrapper()
  if existing then
    existing.value = variable:getValue()  -- update value reference
    return existing
  end

  -- Create new wrapper only if none exists
  local wrapper = {
    variable = variable,
    value = variable:getValue(),
    selectedIndex = 0  -- internal state preserved on reuse
  }
  setmetatable(wrapper, self)
  return wrapper
end
```

## Frontend Library

The frontend library connects to the UI server and supports remote UIs:

**SPA navigation:**
- Binds `historyIndex` and `url`
- When one or both update, triggers `go()` and/or `pushState()` or `replaceState()`

**View rendering:**
- Displays viewdefs when view values change
- The top-level view displays the value of `currentPage()`, a child of variable `1`
- Parses and binds `ui-*` attributes for known widgets
- Values of `ui-*` attributes are paths and can contain property values with URL syntax: `a.b?create=Person&prop=value`
- Properties without values default to `true`: `x?a&b` is equivalent to `x?a=true&b=true`

**Custom widgets (Div):**
- Dynamic Content: `ui-content` attribute - holds HTML
- View: `ui-view` attribute - holds object ref, `ui-namespace` - viewdef namespace
- Dynamic View: `ui-viewdef` attribute - holds computed viewdef
- ViewList: `ui-viewlist` attribute - holds array of object refs, `ui-namespace` - viewdef namespace
  - ViewList uses the `wrapper` variable property to transform domain objects
  - Path properties configure wrapping: `ui-viewlist="contacts?itemWrapper=ContactPresenter"`
  - See viewdefs.md for full ViewList documentation

**Input update behavior:**
- By default, input elements (both native `<input>` and `<sl-input>`) send updates on blur (when the user tabs out)
- Add `keypress` property to send updates on every keypress: `ui-value="name?keypress"`
- This applies to `<input>`, `<textarea>`, `<sl-input>`, and `<sl-textarea>`

**Auto-scroll behavior:**
- Add `scrollOnOutput` property to auto-scroll an element to the bottom when its value updates: `ui-value="log?scrollOnOutput"`
- Useful for log viewers, chat windows, or streaming content containers
- Only scrolls if the element has overflow (is scrollable)

**Shoelace widget bindings:**
- Input: `ui-value`, `ui-disabled`
- Button: `ui-action`
- Select: `ui-items`, `ui-index`, `ui-namespace` - viewdef namespace
