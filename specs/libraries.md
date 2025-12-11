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

**Change detection:**
- Handles detecting and propagating server data changes
- Refresh logic computes values for all watched variables and detects those that have changed
- Does not require support for the observer pattern, allowing any backend object to support variables
- Refreshes happen automatically after receipt of client messages
- Background-triggered changes are throttled
- Provides a thread-safe mechanism for interacting with refresh logic

## Lua Session API

The embedded Lua runtime provides a `session` global for variable management. This is available when `main.lua` executes for each new frontend session.

**Session object:**
```lua
-- Create the app variable (variable 1) - typically done in main.lua
local app = session:createAppVariable(initialValue, properties)

-- Get the app variable (variable 1) after it's created
local app = session:getAppVariable()

-- Create a child variable
local child = session:createVariable(parentId, value, properties)

-- Get a variable by ID
local var = session:getVariable(id)

-- Destroy a variable
session:destroyVariable(id)

-- Watch a property on a variable (react to frontend changes)
session:watchProperty(varId, "propertyName", function(value)
  -- called when property updates from frontend
end)
```

**Built-in property watchers:**

The Lua runtime automatically watches the `lua` property on variable 1. When updated:
- If value ends with `.lua`, loads the file from `<site>/lua/<filename>`
- Otherwise, executes the value as inline Lua code

**Variable wrapper:**
```lua
-- Get variable ID
local id = var:getId()

-- Get current value
local value = var:getValue()

-- Get a property
local prop = var:getProperty("type")

-- Update value and/or properties
var:update(newValue)
var:update(newValue, {type = "Contact"})
var:updateProperties({type = "Contact"})
```

**Viewdef delivery:**

Backends deliver viewdefs by setting the `viewdefs:high` property on variable 1:
```lua
local app = session:getAppVariable()
app:updateProperties({
  ["viewdefs:high"] = {
    ["ContactApp.DEFAULT"] = "<template>...</template>",
    ["Contact.DEFAULT"] = "<template>...</template>"
  }
})
```

**Example main.lua:**
```lua
-- main.lua - Entry point for each new session

-- App presenter with methods callable via ui-action paths
local AppPresenter = {}
AppPresenter.__index = AppPresenter

function AppPresenter:new()
  local self = setmetatable({}, AppPresenter)
  self.items = {}
  return self
end

function AppPresenter:addItem(name)
  local item = session:createVariable(app:getId(), {
    type = "Item",
    name = name
  })
  table.insert(self.items, {obj = item:getId()})
  app:update({items = self.items})
end

function AppPresenter:deleteItem(itemId)
  session:destroyVariable(tonumber(itemId))
  -- Remove from items list...
end

-- Create presenter and app variable
local presenter = AppPresenter:new()
local app = session:createAppVariable({
  type = "App",
  view = "app",
  title = "My Application",
  items = {},
  presenter = presenter  -- ui-action="presenter.addItem(name)" calls this
})
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

**Custom widgets (Div):**
- Dynamic Content: `ui-content` attribute - holds HTML
- View: `ui-view` attribute - holds object ref, `ui-namespace` - viewdef namespace
- Dynamic View: `ui-viewdef` attribute - holds computed viewdef
- ViewList: `ui-viewlist` attribute - holds array of object refs, `ui-namespace` - viewdef namespace

**Shoelace widget bindings:**
- Input: `ui-value`, `ui-disabled`
- Button: `ui-action`
- Select: `ui-items`, `ui-index`, `ui-namespace` - viewdef namespace
