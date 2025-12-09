# Libraries

## Backend Library

The backend library makes integrating with the UI server easy:

**Connection:**
- Connect to UI server with a root value for variable `1`
- Invokes hook upon connection close
- Root value must bind to `currentPage()`
- If the app is an SPA, the frontend will bind to `historyIndex` and `url` on the root

**Path navigation:**
- Handles path navigation with reflection for languages that support it (Go, Python, Julia, JavaScript, Java, Lua, etc.)

**Change detection:**
- Handles detecting and propagating server data changes
- Refresh logic computes values for all watched variables and detects those that have changed
- Does not require support for the observer pattern, allowing any backend object to support variables
- Refreshes happen automatically after receipt of client messages
- Background-triggered changes are throttled
- Provides a thread-safe mechanism for interacting with refresh logic

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
