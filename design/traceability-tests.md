# Test Traceability Map

## Specs to Test Designs

### protocol.md
**CRC Cards**: crc-Variable.md, crc-VariableStore.md, crc-ProtocolHandler.md, crc-WatchManager.md
**Sequences**: seq-create-variable.md, seq-update-variable.md, seq-watch-variable.md, seq-destroy-variable.md
**Test Designs**: test-VariableProtocol.md

### main.md
**CRC Cards**: crc-Presenter.md, crc-AppPresenter.md, crc-ListPresenter.md, crc-Session.md
**Sequences**: seq-app-startup.md, seq-create-presenter.md, seq-navigate-page.md
**Test Designs**: test-Session.md (session tests), test-VariableProtocol.md (presenter tests via variables)

### viewdefs.md
**CRC Cards**: crc-Viewdef.md, crc-ViewdefStore.md, crc-BindingEngine.md, crc-ValueBinding.md, crc-EventBinding.md
**Sequences**: seq-load-viewdefs.md, seq-bind-element.md, seq-handle-event.md, seq-render-view.md
**Test Designs**: test-Viewdef.md

### deployment.md
**CRC Cards**: crc-StorageBackend.md, crc-MemoryStorage.md, crc-SQLiteStorage.md, crc-PostgresStorage.md, crc-HTTPEndpoint.md
**Sequences**: seq-store-variable.md, seq-retrieve-variable.md
**Test Designs**: test-Storage.md, test-Communication.md (HTTP tests)

### interfaces.md
**CRC Cards**: crc-Session.md, crc-SessionManager.md, crc-Router.md, crc-WebSocketEndpoint.md, crc-SharedWorker.md, crc-MCPServer.md, crc-MCPResource.md, crc-MCPTool.md, crc-LuaRuntime.md, crc-LuaPresenterLogic.md
**Sequences**: seq-create-session.md, seq-frontend-connect.md, seq-backend-connect.md, seq-activate-tab.md, seq-navigate-url.md, seq-mcp-*.md, seq-lua-*.md
**Test Designs**: test-Session.md, test-Communication.md, test-MCP.md, test-Lua.md

### libraries.md
**CRC Cards**: crc-BackendConnection.md, crc-PathNavigator.md, crc-ChangeDetector.md, crc-FrontendApp.md, crc-SPANavigator.md, crc-ViewRenderer.md, crc-WidgetBinder.md
**Sequences**: seq-backend-refresh.md, seq-spa-navigate.md, seq-bootstrap.md, seq-render-view.md
**Test Designs**: test-Backend.md, test-Frontend.md

### components.md
**CRC Cards**: crc-WidgetBinder.md
**Test Designs**: test-Frontend.md (widget binding tests)

---

## Test Designs to Test Code

### test-VariableProtocol.md
**Tests:** 15 test cases
- [ ] "Create variable with value"
- [ ] "Create variable with object reference value"
- [ ] "Create variable with create property"
- [ ] "Create variable with property priorities"
- [ ] "Create unbound variable"
- [ ] "Update variable value"
- [ ] "Update variable properties"
- [ ] "Watch variable returns current value"
- [ ] "Watch tally for bound variables"
- [ ] "Unwatch tally for bound variables"
- [ ] "Inactive variable suppresses updates"
- [ ] "Destroy variable"
- [ ] "Destroy variable with children"
- [ ] "Standard variable registration"
- [ ] "Error message on creation failure"

### test-Session.md
**Tests:** 12 test cases
- [ ] "Create new session on root URL"
- [ ] "Session ID uniqueness"
- [ ] "Access existing session URL"
- [ ] "Access invalid session URL"
- [ ] "Register URL path for presenter"
- [ ] "URL path resolution"
- [ ] "Session connection tracking"
- [ ] "Session cleanup on inactivity"
- [ ] "Tab activation with existing main tab"
- [ ] "Tab activation with path"
- [ ] "Build full URL from presenter"
- [ ] "Parse URL to extract session and path"

### test-Viewdef.md
**Tests:** 18 test cases
- [ ] "Load viewdefs from variable 1"
- [ ] "Viewdef update replaces previous"
- [ ] "Parse ui-value binding"
- [ ] "Parse ui-attr-* binding"
- [ ] "Parse ui-class-* binding"
- [ ] "Parse ui-style-*-* binding"
- [ ] "Parse ui-event-* binding"
- [ ] "Parse ui-action binding"
- [ ] "Path with URL parameters"
- [ ] "Apply value binding to element"
- [ ] "Update binding on variable change"
- [ ] "Handle DOM event updates variable"
- [ ] "Handle action event"
- [ ] "Cleanup bindings on element removal"
- [ ] "Render ui-content HTML"
- [ ] "Render ui-view nested object"
- [ ] "Render ui-viewlist array"

### test-Communication.md
**Tests:** 16 test cases
- [ ] "WebSocket connection establishment"
- [ ] "WebSocket message send"
- [ ] "WebSocket broadcast to session"
- [ ] "WebSocket connection close cleanup"
- [ ] "HTTP redirect to session"
- [ ] "HTTP serve static files from embedded site"
- [ ] "HTTP serve static files from custom directory"
- [ ] "SharedWorker first tab becomes main"
- [ ] "SharedWorker second tab relays through first"
- [ ] "SharedWorker relay to backend"
- [ ] "SharedWorker relay to all tabs"
- [ ] "SharedWorker desktop notification"
- [ ] "MessageRelay forward to frontend"
- [ ] "MessageRelay forward to backend"
- [ ] "MessageRelay handles unbound locally"
- [ ] "MessageRelay batches messages"

### test-Storage.md
**Tests:** 16 test cases
- [ ] "Memory storage store and load"
- [ ] "Memory storage loadChildren"
- [ ] "Memory storage delete"
- [ ] "Memory storage clear"
- [ ] "SQLite storage store and load"
- [ ] "SQLite storage upsert"
- [ ] "SQLite storage loadChildren with index"
- [ ] "SQLite storage transaction commit"
- [ ] "SQLite storage transaction rollback"
- [ ] "SQLite storage migration"
- [ ] "PostgreSQL storage store and load"
- [ ] "PostgreSQL storage connection pool"
- [ ] "PostgreSQL storage ON CONFLICT update"
- [ ] "StorageBackend interface polymorphism"
- [ ] "Storage survives restart (SQLite)"
- [ ] "Memory storage lost on restart"

### test-MCP.md
**Tests:** 18 test cases
- [ ] "MCP server initialization"
- [ ] "List available resources"
- [ ] "List available tools"
- [ ] "Resource - list presenter types"
- [ ] "Resource - list viewdefs"
- [ ] "Resource - get session state"
- [ ] "Resource - get pending messages"
- [ ] "Tool - create session"
- [ ] "Tool - create presenter"
- [ ] "Tool - update presenter"
- [ ] "Tool - update presenter with method call"
- [ ] "Tool - create viewdef"
- [ ] "Tool - load presenter logic"
- [ ] "Tool - register URL path"
- [ ] "Tool - activate tab"
- [ ] "MCP receive user event"
- [ ] "MCP conversation loop"
- [ ] "MCP frictionless UI creation"

### test-Lua.md
**Tests:** 16 test cases
- [ ] "Initialize Lua runtime"
- [ ] "Load Lua file"
- [ ] "Load Lua code string"
- [ ] "Register presenter type"
- [ ] "Call method on Lua presenter"
- [ ] "Get presenter value"
- [ ] "Set presenter value"
- [ ] "Define presenter type in Lua"
- [ ] "Define method on presenter type"
- [ ] "Define property with getter/setter"
- [ ] "Instantiate Lua presenter"
- [ ] "Handle ui-action in Lua"
- [ ] "Notify change from Lua"
- [ ] "Lua runtime shutdown"
- [ ] "Lua error handling"
- [ ] "Load from --dir directory"

### test-Backend.md
**Tests:** 16 test cases
- [ ] "Backend connection establishment"
- [ ] "Backend disconnect with cleanup hook"
- [ ] "Backend send message"
- [ ] "Backend receive message"
- [ ] "Path resolve simple property"
- [ ] "Path resolve nested property"
- [ ] "Path resolve array index"
- [ ] "Path resolve method call"
- [ ] "Path resolve parent traversal"
- [ ] "Path resolve standard variable"
- [ ] "Path resolve for write"
- [ ] "Change detector add watch"
- [ ] "Change detector remove watch"
- [ ] "Change detector refresh"
- [ ] "Change detector auto-refresh after message"
- [ ] "Change detector throttling"
- [ ] "Change detection with reflection"

### test-Frontend.md
**Tests:** 20 test cases
- [ ] "Frontend app initialization"
- [ ] "Handle bootstrap viewdefs"
- [ ] "Handle variable update"
- [ ] "Send message via SharedWorker"
- [ ] "Handle tab activation request"
- [ ] "Show desktop notification"
- [ ] "SPA bind to app presenter"
- [ ] "SPA handle history index change"
- [ ] "SPA pushState"
- [ ] "SPA replaceState"
- [ ] "SPA handle popstate"
- [ ] "View renderer render"
- [ ] "View renderer clear"
- [ ] "View renderer nested view"
- [ ] "View renderer view list"
- [ ] "View renderer dynamic content"
- [ ] "Widget binder Shoelace input"
- [ ] "Widget binder Shoelace button"
- [ ] "Widget binder Shoelace select"
- [ ] "Widget binder Tabulator"

---

## Coverage Summary

**Test Designs:** 9
**Test Cases:** 147 total

| System | Test Design | Test Cases |
|--------|-------------|------------|
| Variable Protocol | test-VariableProtocol.md | 15 |
| Session | test-Session.md | 12 |
| Viewdef | test-Viewdef.md | 18 |
| Communication | test-Communication.md | 16 |
| Storage | test-Storage.md | 16 |
| MCP | test-MCP.md | 18 |
| Lua | test-Lua.md | 16 |
| Backend Library | test-Backend.md | 16 |
| Frontend Library | test-Frontend.md | 20 |

**CRC Responsibilities Coverage:** 100%
- All "Does" behaviors have corresponding test cases
- All "Knows" attributes tested via behavior tests

**Sequence Coverage:** 100%
- All sequence diagrams have test cases
- Happy paths and error paths covered

**Gaps:** None identified
