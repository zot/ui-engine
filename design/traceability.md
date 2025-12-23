# Traceability Map

## Level 1 <-> Level 2 (Specs to Design Models)

### main.md

**CRC Cards:**
- crc-Presenter.md
- crc-AppPresenter.md
- crc-ListPresenter.md
- crc-Session.md
- crc-Backend.md (UI Server Architecture section)
- crc-LuaBackend.md (UI Server Architecture - Hosted Backend)

**Sequence Diagrams:**
- seq-app-startup.md
- seq-create-presenter.md
- seq-navigate-page.md
- seq-session-create-backend.md

**UI Specs:**
- ui-app-shell.md

---

### protocol.md

**CRC Cards:**
- crc-Variable.md
- crc-VariableStore.md
- crc-ProtocolHandler.md
- crc-Wrapper.md
- crc-ObjectReference.md
- crc-PathSyntax.md
- crc-MessageBatcher.md
- crc-LuaBackend.md (Session-Based Communication)

**Sequence Diagrams:**
- seq-create-variable.md
- seq-update-variable.md
- seq-watch-variable.md
- seq-destroy-variable.md
- seq-wrapper-transform.md
- seq-relay-message.md
- seq-path-resolve.md
- seq-viewdef-delivery.md
- seq-backend-watch.md
- seq-backend-detect-changes.md

**Notes:**
- WatchManager removed - watch functionality merged into LuaBackend (per-session)

---

### viewdefs.md

**CRC Cards:**
- crc-Viewdef.md
- crc-ViewdefStore.md
- crc-View.md
- crc-ViewList.md
- crc-ViewListItem.md
- crc-AppView.md
- crc-BindingEngine.md
- crc-ValueBinding.md
- crc-EventBinding.md

**Sequence Diagrams:**
- seq-load-viewdefs.md
- seq-viewdef-delivery.md
- seq-render-view.md
- seq-viewlist-update.md
- seq-viewlist-presenter-sync.md
- seq-bind-element.md
- seq-handle-event.md

---

### deployment.md

**CRC Cards:**
- crc-HTTPEndpoint.md
- crc-BackendSocket.md
- crc-ProtocolDetector.md
- crc-PacketProtocol.md
- crc-PendingResponseQueue.md
- crc-Config.md

**Sequence Diagrams:**
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
- crc-LuaRuntime.md
- crc-LuaSession.md
- crc-LuaPresenterLogic.md

**Sequence Diagrams:**
- seq-create-session.md
- seq-session-create-backend.md
- seq-frontend-connect.md
- seq-frontend-reconnect.md
- seq-activate-tab.md
- seq-navigate-url.md
- seq-lua-executor-init.md
- seq-lua-session-init.md
- seq-lua-execute.md
- seq-load-lua-code.md
- seq-lua-handle-action.md

**UI Specs:**
- manifest-ui.md

---

### specs/mcp.md

**CRC Cards:**
- crc-MCPServer.md
- crc-MCPResource.md
- crc-MCPTool.md

**Sequence Diagrams:**
- seq-mcp-lifecycle.md
- seq-mcp-create-session.md
- seq-mcp-create-presenter.md
- seq-mcp-receive-event.md
- seq-mcp-run.md
- seq-mcp-get-state.md

---

### data-models.md

**CRC Cards:**
- crc-VariableStore.md (unbound model)
- crc-BackendConnection.md (bound model)
- crc-ChangeDetector.md

---

### libraries.md

**CRC Cards:**
- crc-PathNavigator.md
- crc-PathSyntax.md (path property defaults)
- crc-ChangeDetector.md
- crc-ObjectRegistry.md
- crc-BackendConnection.md
- crc-FrontendApp.md
- crc-SPANavigator.md
- crc-ViewRenderer.md
- crc-WidgetBinder.md
- crc-BindingEngine.md (input update behavior)
- crc-ValueBinding.md (input event selection)
- crc-MessageRelay.md
- crc-LuaSession.md
- crc-LuaVariable.md

**Sequence Diagrams:**
- seq-spa-navigate.md
- seq-bootstrap.md
- seq-lua-session-init.md
- seq-backend-refresh.md
- seq-object-registry.md
- seq-input-value-binding.md

**Notes:**
- BackendConnection used by external Go backends (connected backend mode)
- Embedded Lua uses LuaSession instead of BackendConnection
- ObjectRegistry provides identity-based serialization for Go backends (requires Go 1.25+)
- Path properties without values default to `true` (e.g., `?keypress` equals `?keypress=true`)
- Input elements use blur-based events by default; `keypress` property switches to keystroke events

---

### components.md

**CRC Cards:**
- crc-WidgetBinder.md

---

## Level 2 <-> Level 3 (Design Models to Implementation)

*Implementation checkboxes updated to reflect actual code*

### crc-Variable.md
**Source Spec:** protocol.md
**Implementation:**
- [x] `internal/variable/variable.go` - Variable struct and methods
- [x] `internal/variable/variable.go` - Add wrapper property, dual value architecture
- [x] `web/src/variable.ts` - Frontend variable representation

### crc-VariableStore.md
**Source Spec:** protocol.md, data-models.md
**Implementation:**
- [x] `internal/variable/store.go` - VariableStore implementation
- [x] `web/src/connection.ts` - Frontend VariableStore class

### crc-ProtocolHandler.md
**Source Spec:** protocol.md
**Implementation:**
- [x] `internal/protocol/handler.go` - Protocol message handling
- [x] `web/src/protocol.ts` - Frontend protocol types and encoding

### crc-Backend.md
**Source Spec:** main.md (UI Server Architecture)
**Implementation:**
- [ ] `internal/backend/backend.go` - Backend interface

### crc-LuaBackend.md
**Source Spec:** main.md (UI Server Architecture), protocol.md (Session-Based Communication)
**Implementation:**
- [x] `internal/backend/lua.go` - LuaBackend with per-session change-tracker

**Notes:**
- Merges WatchManager functionality (watchCounts, watchers maps are per-session)
- Owns change-tracker.Tracker instance (per-session, not global)
- Fixes bug where global WatchManager maps caused variable ID collisions between sessions

### crc-Presenter.md
**Source Spec:** main.md
**Implementation:**
- [x] `internal/presenter/presenter.go` - Base presenter (includes AppPresenter, ListPresenter)
- [x] `lib/presenter.lua` - Lua presenter base

### crc-AppPresenter.md
**Source Spec:** main.md
**Implementation:**
- [x] `internal/presenter/presenter.go` - App presenter (combined with Presenter)
- [x] `lib/app.lua` - Lua app presenter

### crc-ListPresenter.md
**Source Spec:** main.md
**Implementation:**
- [x] `internal/presenter/presenter.go` - List presenter (combined with Presenter)
- [x] `lib/list.lua` - Lua list presenter

### crc-Viewdef.md
**Source Spec:** viewdefs.md
**Implementation:**
- [x] `internal/viewdef/viewdef.go` - Viewdef struct
- [x] `web/src/viewdef.ts` - Frontend viewdef handling

### crc-ViewdefStore.md
**Source Spec:** viewdefs.md
**Implementation:**
- [x] `internal/viewdef/store.go` - Server viewdef store
- [x] `web/src/viewdef_store.ts` - Frontend viewdef cache with validation and pending views

### crc-View.md
**Source Spec:** viewdefs.md
**Implementation:**
- [x] `web/src/view.ts` - View class for ui-view elements

### crc-ViewList.md
**Source Spec:** viewdefs.md, protocol.md
**Implementation:**
- [x] `web/src/viewlist.ts` - ViewList class for ui-viewlist elements (frontend)
- [x] `internal/lua/viewlist.go` - ViewList wrapper (backend)

### crc-ViewListItem.md
**Source Spec:** viewdefs.md
**Implementation:**
- [x] `internal/lua/viewlistitem.go` - ViewListItem struct (item, list, index)

### crc-AppView.md
**Source Spec:** viewdefs.md
**Implementation:**
- [x] `web/src/app_view.ts` - AppView class for ui-app element

### crc-BindingEngine.md
**Source Spec:** viewdefs.md, libraries.md
**Implementation:**
- [x] `web/src/binding.ts` - Binding engine with child variable architecture (all bindings create child variables for server-side path resolution)

### crc-ValueBinding.md
**Source Spec:** viewdefs.md, libraries.md
**Implementation:**
- [x] `web/src/binding.ts` - Value bindings with child variable creation, event selection based on keypress property

### crc-EventBinding.md
**Source Spec:** viewdefs.md
**Implementation:**
- [x] `web/src/binding.ts` - Event bindings (combined with BindingEngine)

### crc-Session.md
**Source Spec:** main.md (UI Server Architecture - Frontend Layer), interfaces.md
**Implementation:**
- [x] `internal/session/session.go` - Session struct
- [x] `internal/session/session.go` - backend field, delegate to Backend

### crc-SessionManager.md
**Source Spec:** interfaces.md, protocol.md
**Implementation:**
- [x] `internal/session/manager.go` - Session management with vended ID mapping

### crc-Router.md
**Source Spec:** interfaces.md
**Implementation:**
- [x] `internal/router/router.go` - URL routing
- [x] `web/src/router.ts` - Frontend routing

### crc-WebSocketEndpoint.md
**Source Spec:** interfaces.md, deployment.md
**Implementation:**
- [x] `internal/server/websocket.go` - WebSocket handler
- [x] `web/src/connection.ts` - Frontend Connection class

### crc-HTTPEndpoint.md
**Source Spec:** interfaces.md, deployment.md
**Implementation:**
- [x] `internal/server/http.go` - HTTP handler

### crc-SharedWorker.md
**Source Spec:** interfaces.md
**Implementation:**
- [x] `web/src/worker.ts` - SharedWorker

### crc-MessageRelay.md
**Source Spec:** protocol.md
**Implementation:**
- [x] `internal/server/relay.go` - Message relay

### crc-MessageBatcher.md
**Source Spec:** protocol.md
**Implementation:**
- [x] `internal/protocol/batcher.go` - Priority-based message batching
- [x] `web/src/batcher.ts` - Frontend batch processing

### crc-MCPServer.md
**Source Spec:** specs/mcp.md
**Implementation:**
- [x] `internal/mcp/server.go` - MCP server (Lifecycle FSM, configuration, startup, notifications)

### crc-MCPResource.md
**Source Spec:** specs/mcp.md
**Implementation:**
- [x] `internal/mcp/resources.go` - MCP resources (State root redirection via mcp.state)

### crc-MCPTool.md
**Source Spec:** specs/mcp.md
**Implementation:**
- [x] `internal/mcp/tools.go` - MCP tools (configure, start, run, upload_viewdef, open_browser)

### crc-LuaRuntime.md
**Source Spec:** interfaces.md, deployment.md, specs/mcp.md
**Implementation:**
- [x] `internal/lua/runtime.go` - Lua runtime with session API, I/O redirection, and mcp global

### crc-LuaSession.md
**Source Spec:** libraries.md, interfaces.md, protocol.md, specs/mcp.md
**Implementation:**
- [x] `internal/lua/runtime.go` - Go-side LuaSession implementation (uses vended session IDs)
- [x] `internal/lua/runtime.go` - Automatic change detection via AfterBatch()
- [x] `internal/lua/runtime.go` - Support for mcp.state and mcp.notify

### crc-LuaVariable.md
**Source Spec:** libraries.md
**Implementation:**
- [x] `internal/lua/runtime.go` - Object reference tracking (for automatic change detection)

### crc-LuaPresenterLogic.md
**Source Spec:** interfaces.md
**Implementation:**
- [x] `lib/presenter_logic.lua` - Presenter logic base

### crc-BackendConnection.md
**Source Spec:** libraries.md
**Notes:** Used by external Go backends (connected backend mode); embedded Lua uses LuaSession instead
**Implementation:**
- [x] `lib/go/connection.go` - Go backend connection
- [ ] `lib/lua/connection.lua` - (not needed - embedded Lua uses LuaSession)

### crc-PathNavigator.md
**Source Spec:** protocol.md, libraries.md
**Implementation:**
- [x] `lib/go/path.go` - Go path navigation
- [x] `lib/lua/path.lua` - Lua path navigation
- [x] `web/src/path.ts` - Frontend path resolution

### crc-ChangeDetector.md
**Source Spec:** libraries.md
**Implementation:**
- [x] `lib/go/change.go` - Go change detection
- [x] `lib/lua/change.lua` - Lua change detection (to be removed - superseded by Go change-tracker)

### crc-ObjectRegistry.md
**Source Spec:** libraries.md
**Notes:** Go 1.25+ required for weak package; provides identity-based serialization
**Implementation:**
- [ ] `lib/go/registry.go` - Object registry with weak references

### crc-FrontendApp.md
**Source Spec:** libraries.md, interfaces.md
**Implementation:**
- [x] `web/src/app.ts` - Frontend app (includes SPA navigation)

### crc-SPANavigator.md
**Source Spec:** libraries.md, interfaces.md
**Implementation:**
- [x] `web/src/app.ts` - SPA navigation (combined with FrontendApp)

### crc-ViewRenderer.md
**Source Spec:** libraries.md
**Implementation:**
- [x] `web/src/renderer.ts` - View renderer

### crc-WidgetBinder.md
**Source Spec:** libraries.md, components.md
**Implementation:**
- [x] `web/src/binding.ts` - Widget bindings for Shoelace inputs (sl-input/sl-textarea event selection based on keypress) integrated into BindingEngine

### crc-ObjectReference.md
**Source Spec:** protocol.md
**Implementation:**
- [x] `internal/variable/variable.go` - Object reference handling (in Variable)
- [x] `web/src/variable.ts` - Frontend ObjectReference type

### crc-PathSyntax.md
**Source Spec:** protocol.md, viewdefs.md, libraries.md
**Implementation:**
- [x] `internal/path/syntax.go` - Path parsing
- [x] `web/src/binding.ts` - Frontend path parsing (properties without values default to true)

### crc-BackendSocket.md
**Source Spec:** deployment.md, interfaces.md
**Notes:** Supports connected backend only or hybrid mode (with embedded Lua)
**Implementation:**
- [x] `internal/server/backend_socket.go` - Backend socket handler

### crc-PendingResponseQueue.md
**Source Spec:** deployment.md
**Implementation:**
- [x] `internal/server/pending.go` - Pending response queue

### crc-Config.md
**Source Spec:** deployment.md
**Implementation:**
- [x] `internal/config/config.go` - Configuration loading

### crc-Wrapper.md
**Source Spec:** protocol.md
**Implementation:**
- [x] `internal/lua/wrapper.go` - Wrapper interface and registry
- [x] `internal/lua/viewlist.go` - ViewList wrapper implementation
- [ ] `lib/wrapper.lua` - Lua wrapper base (optional - Go implementation complete)

---

## Removed Design Elements

### crc-WatchManager.md (REMOVED)
**Status:** Functionality merged into crc-LuaBackend.md
**Reason:** WatchManager used global maps keyed by varID, but variable IDs are only unique within a session. Per-session watch management is now handled by LuaBackend.
**Implementation to remove:**
- [ ] `internal/variable/watch.go` - WatchManager (remove)
- [ ] `lib/lua/change.lua` - Lua change detection (remove - superseded by Go change-tracker)
