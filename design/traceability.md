# Traceability Map

## Level 1 <-> Level 2 (Specs to Design Models)

### main.md

**CRC Cards:**
- crc-Presenter.md
- crc-AppPresenter.md
- crc-ListPresenter.md
- crc-Session.md

**Sequence Diagrams:**
- seq-app-startup.md
- seq-create-presenter.md
- seq-navigate-page.md

**UI Specs:**
- ui-app-shell.md

---

### protocol.md

**CRC Cards:**
- crc-Variable.md
- crc-VariableStore.md
- crc-ProtocolHandler.md
- crc-WatchManager.md
- crc-ObjectReference.md
- crc-PathSyntax.md

**Sequence Diagrams:**
- seq-create-variable.md
- seq-update-variable.md
- seq-watch-variable.md
- seq-destroy-variable.md
- seq-relay-message.md
- seq-path-resolve.md

---

### viewdefs.md

**CRC Cards:**
- crc-Viewdef.md
- crc-ViewdefStore.md
- crc-BindingEngine.md
- crc-ValueBinding.md
- crc-EventBinding.md

**Sequence Diagrams:**
- seq-load-viewdefs.md
- seq-bind-element.md
- seq-handle-event.md
- seq-render-view.md

---

### deployment.md

**CRC Cards:**
- crc-StorageBackend.md
- crc-MemoryStorage.md
- crc-SQLiteStorage.md
- crc-PostgresStorage.md
- crc-HTTPEndpoint.md
- crc-BackendSocket.md
- crc-ProtocolDetector.md
- crc-PacketProtocol.md
- crc-PendingResponseQueue.md
- crc-Config.md

**Sequence Diagrams:**
- seq-store-variable.md
- seq-retrieve-variable.md
- seq-backend-socket-accept.md
- seq-poll-pending.md
- seq-server-startup.md

---

### interfaces.md

**CRC Cards:**
- crc-Session.md
- crc-SessionManager.md
- crc-Router.md
- crc-WebSocketEndpoint.md
- crc-SharedWorker.md
- crc-MCPServer.md
- crc-MCPResource.md
- crc-MCPTool.md
- crc-LuaRuntime.md
- crc-LuaPresenterLogic.md

**Sequence Diagrams:**
- seq-create-session.md
- seq-frontend-connect.md
- seq-frontend-reconnect.md
- seq-backend-connect.md
- seq-activate-tab.md
- seq-navigate-url.md
- seq-mcp-create-session.md
- seq-mcp-create-presenter.md
- seq-mcp-receive-event.md
- seq-load-lua-code.md
- seq-lua-handle-action.md

**UI Specs:**
- manifest-ui.md

---

### data-models.md

**CRC Cards:**
- crc-VariableStore.md (unbound model)
- crc-BackendConnection.md (bound model)
- crc-ChangeDetector.md

---

### libraries.md

**CRC Cards:**
- crc-BackendConnection.md
- crc-PathNavigator.md
- crc-ChangeDetector.md
- crc-FrontendApp.md
- crc-SPANavigator.md
- crc-ViewRenderer.md
- crc-WidgetBinder.md
- crc-MessageRelay.md

**Sequence Diagrams:**
- seq-backend-refresh.md
- seq-spa-navigate.md
- seq-bootstrap.md

---

### components.md

**CRC Cards:**
- crc-WidgetBinder.md

---

## Level 2 <-> Level 3 (Design Models to Implementation)

*Implementation checkboxes to be filled as code is written*

### crc-Variable.md
**Source Spec:** protocol.md
**Implementation:**
- [ ] `server/variable.go` - Variable struct and methods
- [ ] `frontend/variable.ts` - Frontend variable representation

### crc-VariableStore.md
**Source Spec:** protocol.md, data-models.md
**Implementation:**
- [ ] `server/store.go` - VariableStore implementation

### crc-ProtocolHandler.md
**Source Spec:** protocol.md
**Implementation:**
- [ ] `server/protocol.go` - Protocol message handling

### crc-WatchManager.md
**Source Spec:** protocol.md
**Implementation:**
- [ ] `server/watch.go` - Watch subscription management

### crc-Presenter.md
**Source Spec:** main.md
**Implementation:**
- [ ] `server/presenter.go` - Base presenter
- [ ] `lib/presenter.lua` - Lua presenter base

### crc-AppPresenter.md
**Source Spec:** main.md
**Implementation:**
- [ ] `server/app_presenter.go` - App presenter
- [ ] `lib/app.lua` - Lua app presenter

### crc-ListPresenter.md
**Source Spec:** main.md
**Implementation:**
- [ ] `server/list_presenter.go` - List presenter
- [ ] `lib/list.lua` - Lua list presenter

### crc-Viewdef.md
**Source Spec:** viewdefs.md
**Implementation:**
- [ ] `server/viewdef.go` - Viewdef struct
- [ ] `frontend/viewdef.ts` - Frontend viewdef handling

### crc-ViewdefStore.md
**Source Spec:** viewdefs.md
**Implementation:**
- [ ] `server/viewdef_store.go` - Server viewdef store
- [ ] `frontend/viewdef_store.ts` - Frontend viewdef cache

### crc-BindingEngine.md
**Source Spec:** viewdefs.md, libraries.md
**Implementation:**
- [ ] `frontend/binding.ts` - Binding engine

### crc-ValueBinding.md
**Source Spec:** viewdefs.md
**Implementation:**
- [ ] `frontend/value_binding.ts` - Value bindings

### crc-EventBinding.md
**Source Spec:** viewdefs.md
**Implementation:**
- [ ] `frontend/event_binding.ts` - Event bindings

### crc-Session.md
**Source Spec:** main.md, interfaces.md
**Implementation:**
- [ ] `server/session.go` - Session struct

### crc-SessionManager.md
**Source Spec:** interfaces.md
**Implementation:**
- [ ] `server/session_manager.go` - Session management

### crc-Router.md
**Source Spec:** interfaces.md
**Implementation:**
- [ ] `server/router.go` - URL routing
- [ ] `frontend/router.ts` - Frontend routing

### crc-WebSocketEndpoint.md
**Source Spec:** interfaces.md, deployment.md
**Implementation:**
- [ ] `server/websocket.go` - WebSocket handler

### crc-HTTPEndpoint.md
**Source Spec:** interfaces.md, deployment.md
**Implementation:**
- [ ] `server/http.go` - HTTP handler

### crc-SharedWorker.md
**Source Spec:** interfaces.md
**Implementation:**
- [ ] `frontend/worker.ts` - SharedWorker

### crc-MessageRelay.md
**Source Spec:** protocol.md
**Implementation:**
- [ ] `server/relay.go` - Message relay

### crc-StorageBackend.md
**Source Spec:** deployment.md, data-models.md
**Implementation:**
- [ ] `server/storage.go` - Storage interface

### crc-MemoryStorage.md
**Source Spec:** deployment.md
**Implementation:**
- [ ] `server/memory_storage.go` - In-memory storage

### crc-SQLiteStorage.md
**Source Spec:** deployment.md
**Implementation:**
- [ ] `server/sqlite_storage.go` - SQLite storage

### crc-PostgresStorage.md
**Source Spec:** deployment.md
**Implementation:**
- [ ] `server/postgres_storage.go` - PostgreSQL storage

### crc-MCPServer.md
**Source Spec:** interfaces.md
**Implementation:**
- [ ] `mcp/server.go` - MCP server

### crc-MCPResource.md
**Source Spec:** interfaces.md
**Implementation:**
- [ ] `mcp/resources.go` - MCP resources

### crc-MCPTool.md
**Source Spec:** interfaces.md
**Implementation:**
- [ ] `mcp/tools.go` - MCP tools

### crc-LuaRuntime.md
**Source Spec:** interfaces.md, deployment.md
**Implementation:**
- [ ] `server/lua.go` - Lua runtime

### crc-LuaPresenterLogic.md
**Source Spec:** interfaces.md
**Implementation:**
- [ ] `lib/presenter_logic.lua` - Presenter logic base

### crc-BackendConnection.md
**Source Spec:** libraries.md
**Implementation:**
- [ ] `lib/go/connection.go` - Go backend connection
- [ ] `lib/connection.lua` - Lua backend connection

### crc-PathNavigator.md
**Source Spec:** protocol.md, libraries.md
**Implementation:**
- [ ] `lib/go/path.go` - Go path navigation
- [ ] `lib/path.lua` - Lua path navigation
- [ ] `frontend/path.ts` - Frontend path resolution

### crc-ChangeDetector.md
**Source Spec:** libraries.md
**Implementation:**
- [ ] `lib/go/change.go` - Go change detection
- [ ] `lib/change.lua` - Lua change detection

### crc-FrontendApp.md
**Source Spec:** libraries.md, interfaces.md
**Implementation:**
- [ ] `frontend/app.ts` - Frontend app

### crc-SPANavigator.md
**Source Spec:** libraries.md, interfaces.md
**Implementation:**
- [ ] `frontend/navigator.ts` - SPA navigation

### crc-ViewRenderer.md
**Source Spec:** libraries.md
**Implementation:**
- [ ] `frontend/renderer.ts` - View renderer

### crc-WidgetBinder.md
**Source Spec:** libraries.md, components.md
**Implementation:**
- [ ] `frontend/widgets.ts` - Widget bindings

### crc-ObjectReference.md
**Source Spec:** protocol.md
**Implementation:**
- [ ] `server/object_ref.go` - Object reference handling

### crc-PathSyntax.md
**Source Spec:** protocol.md, viewdefs.md
**Implementation:**
- [ ] `server/path_syntax.go` - Path parsing
- [ ] `frontend/path_syntax.ts` - Frontend path parsing

### crc-BackendSocket.md
**Source Spec:** deployment.md
**Implementation:**
- [ ] `server/backend_socket.go` - Backend socket listener

### crc-ProtocolDetector.md
**Source Spec:** deployment.md
**Implementation:**
- [ ] `server/protocol_detector.go` - Protocol detection

### crc-PacketProtocol.md
**Source Spec:** deployment.md
**Implementation:**
- [ ] `server/packet_protocol.go` - Packet protocol handler

### crc-PendingResponseQueue.md
**Source Spec:** deployment.md
**Implementation:**
- [ ] `server/pending_queue.go` - Pending response queue
