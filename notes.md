# Current Items

HandleFrontEndCreate and HandleFrontEndUpdate need sessionIds



# to do

Audit the implementation using the new Core Principles


We need to merge WatchManager into Session and give each session its own change tracker.
Let's look at what that means for the specs, design, and implementation

There are two conceptual parts of the ui server:
- the frontend connection which manages sessions an routes messages to the backend connection
- the backend connection which either
  - Hosted: manages a Lua backend for each session
    - manages variables and track changes
  - Proxied: sends messages to and from a connected backend
    - does not manage variables, just relays messages

# notes

Make a --diag option that adds a variable analyzer at `/diag` that shows the variable tree and updates it as it changes


we need to switch to a Go Lua implementation that supports weak references. Maybe use the C version of Lua.


- Factor out common Lua code that can work with both external and embedded Lua.
- Make sure that actions in Lua are done reflectively



modify spec and design: the MCP should provide guidance to Claude on how to make viewdefs and make presenters in Lua



# Backend Library

The backend library makes integrating with the UI server easy:
- connect to UI server with a root value for variable `1`
  - invokes hook upon connection close
  - root value must bind to `currentPage()`
  - if the app is to be an SPA app, the frontend will successfully bind to `historyIndex` and `url` on the root
- handles path navigation with reflection for languages that support it (Go, Python, Julia, JavaScript, Java, Lua, and so on)
- handles detecting and propagating server data changes
  - refrech logic computes values for all watched variables and detects those that have changed
  - does not require support for the observer pattern, thus allowing any backend object to support variables
  - refreshes happen automatically after receipt of client messages and are throttled in the case of background-triggered changes
  - provides a thread-safe mechanism for interacting with refresh logic

# Frontend Library

The frontend library connects to the UI server and supports remote UIs
- binds `historyIndex` and `url`
  - when one or both update, it triggers a `go()` and/or `pushState()` or `replaceState()`
- displays viewdefs when view values change
  - the top-level view displays the value of `currentPage()`, a child of variable `1`
- parses and binds `ui-*` attributes for known widgets
  - values of `ui-*` attributes are paths and can contain property values with URL syntax: `a.b?create=Person&maluba=x`
- custom widgets
  - Div
    - Dynamic Content: `ui-content` attribute -- holds HTML
    - View: `ui-view` attribute -- holds object ref, `ui-namespace` -- viewdef namespace
      - Dynamic View: `ui-viewdef` attribute -- holds computed viewdef
    - ViewList: `ui-viewlist` attribute -- holds array of object refs, `ui-namespace` -- viewdef namespace
- Shoelace widgets
  - Input: `ui-value`, `ui-disabled`
  - Button: `ui-action`
  - Select: `ui-items`, `ui-index`, `ui-namespace` -- viewdef namespace

# UI server code
The UI server manages some standard presenters, provided in Go. Developer can provide gopher Lua presenters in the `lua` directory

App Presenter:
- currentPage(): returns history[historyIndex]
- url
- historyIndex
- history: array of objects (set by backend)

List Presenter:
- items: the objects
- selectionIndex: index of selected item
- disabled: defaults to false
