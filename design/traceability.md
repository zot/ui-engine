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
- crc-FrontendOutgoingBatcher.md (Frontend outgoing batching)
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
- seq-frontend-outgoing-batch.md (Frontend throttled batching with priority)

**Notes:**
- WatchManager removed - watch functionality merged into LuaBackend (per-session)

---

### viewdefs.md

**CRC Cards:**
- crc-Viewdef.md
- crc-ViewdefStore.md
- crc-View.md (includes 3-tier namespace resolution, namespace property inheritance, default access=r, elementId)
- crc-ViewList.md (includes fallbackNamespace setting, exemplar namespace inheritance, default access=r, elementId)
- crc-ViewListItem.md
- crc-AppView.md (elementId instead of element)
- crc-Widget.md (binding context for elements with ui-* bindings, uses ElementIdVendor, hasBinding for sibling lookup, unbindHandlers map, unbindAll())
- crc-BindingEngine.md (includes ui-code binding, Widget management, default access property logic, widgets map keyed by elementId, local value setting, sendVariableUpdate, Widget-based binding ownership, duplicate update suppression)
- crc-ValueBinding.md (includes code binding execution with extended scope, default access property, duplicate update suppression)
- crc-EventBinding.md (elementId instead of element, widget reference, syncValueBinding, event update behavior with duplicate suppression, unbind handler registration)
- crc-ElementIdVendor.md (global vendor for element IDs - cross-cutting)

**Sequence Diagrams:**
- seq-load-viewdefs.md
- seq-viewdef-delivery.md
- seq-render-view.md (includes 3-tier namespace resolution)
- seq-viewlist-update.md (includes exemplar namespace inheritance)
- seq-viewlist-presenter-sync.md
- seq-bind-element.md (includes Widget creation via ElementIdVendor, ui-code binding, default access property)
- seq-handle-event.md (includes value sync and local value setting)
- seq-handle-keypress-event.md (ui-event-keypress-* specific key detection)

**Notes:**
- **No Direct Element References (Cross-Cutting Requirement)**: Frontend code MUST NOT store direct references to DOM elements
- **Widget-Based Binding Ownership**: Widget is sole owner of all bindings for an element (no separate Binding interface)
- Widget: Binding context with elementId, variable map, and unbindHandlers map
- Widget.unbindAll() calls all cleanup handlers; BindingEngine.unbindElement() calls widget.unbindAll()
- Variables store elementId reference to Widget (not direct DOM references)
- Element ID format: `ui-{counter}` (global ElementIdVendor, counter starts at 1)
- Namespace resolution: namespace -> fallbackNamespace -> DEFAULT
- ViewList wrapper sets `fallbackNamespace: "list-item"` on its variable
- Default access=r for: ui-value on non-interactive elements, ui-attr-*, ui-class-*, ui-style-*, ui-code, ui-view, ui-viewlist
- ui-code binding executes JavaScript code with element, value, variable, and store in scope
- ui-event-keypress-* bindings listen for specific keys (with optional modifiers) and set variable to key name (e.g., "enter")
- Keypress modifier support: `ui-event-keypress-{modifiers}-{key}` where modifiers can be ctrl, shift, alt, meta (combinable)
- Exact modifier matching: specified modifiers must all be pressed, no additional modifiers allowed
- **Frontend Update Behavior (Universal)**: When sending ANY variable update to backend, MUST first set value in local variable cache
- **Duplicate Update Suppression**: Bindings without `access=action` or `access=w` MUST NOT send update if value unchanged
- **Event binding value sync**: Before sending event update, check for ui-value binding on same widget; if element value differs from cached variable value, send value update first (subject to duplicate suppression)
- **Auto-scroll on output**: `scrollOnOutput` path property auto-scrolls element to bottom when value updates (for log viewers, chat windows, etc.)

---

### deployment.md

**CRC Cards:**
- crc-HTTPEndpoint.md
- crc-BackendSocket.md
- crc-ProtocolDetector.md
- crc-PacketProtocol.md
- crc-PendingResponseQueue.md
- crc-Config.md
- crc-LuaHotLoader.md (Lua Hot-Loading section)

**Sequence Diagrams:**
- seq-backend-socket-accept.md
- seq-poll-pending.md
- seq-server-startup.md
- seq-lua-hotload.md (Lua Hot-Loading section)

---

### interfaces.md

**CRC Cards:**
- crc-Session.md
- crc-SessionManager.md
- crc-Router.md
- crc-WebSocketEndpoint.md
- crc-SharedWorker.md
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

### data-models.md

**CRC Cards:**
- crc-VariableStore.md (unbound model)
- crc-BackendConnection.md (bound model)

**External Package:** Change detection provided by `change-tracker` (`github.com/zot/change-tracker`)

---

### libraries.md

**External Package:** Core tracking provided by `change-tracker` (`github.com/zot/change-tracker`)
- Variable management, change detection, object registry, value serialization

**CRC Cards:**
- crc-PathNavigator.md
- crc-PathSyntax.md (path property defaults)
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
- seq-input-value-binding.md

**Notes:**
- BackendConnection used by external Go backends (connected backend mode)
- Embedded Lua uses LuaSession instead of BackendConnection
- Path properties without values default to `true` (e.g., `?keypress` equals `?keypress=true`)
- Input elements use blur-based events by default; `keypress` property switches to keystroke events
- `scrollOnOutput` path property auto-scrolls element to bottom on value update (for log viewers, chat windows)

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
- [x] `internal/variable/variable.go` - namespace/fallbackNamespace property handling (generic properties)
- [x] `web/src/variable.ts` - Frontend variable representation
- [x] `web/src/variable.ts` - namespace/fallbackNamespace property inheritance (via view.ts, viewlist.ts)

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
- [x] `internal/backend/backend.go` - Backend interface

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
- [x] `web/src/view.ts` - 3-tier namespace resolution (namespace -> fallbackNamespace -> DEFAULT)
- [x] `web/src/view.ts` - Namespace property inheritance from parent variable
- [x] `web/src/view.ts` - Uses elementId (getElement via document.getElementById)

### crc-ViewList.md
**Source Spec:** viewdefs.md, protocol.md
**Implementation:**
- [x] `web/src/viewlist.ts` - ViewList class for ui-viewlist elements (frontend)
- [x] `web/src/viewlist.ts` - Exemplar namespace inheritance (fallbackNamespace)
- [x] `web/src/viewlist.ts` - Uses elementId and viewIds (getElement via document.getElementById)
- [x] `internal/lua/viewlist.go` - ViewList wrapper (backend)
- [x] `internal/lua/viewlist.go` - Sets fallbackNamespace:high to "list-item"

### crc-ViewListItem.md
**Source Spec:** viewdefs.md
**Implementation:**
- [x] `internal/lua/viewlistitem.go` - ViewListItem struct (item, list, index)

### crc-AppView.md
**Source Spec:** viewdefs.md
**Implementation:**
- [x] `web/src/app_view.ts` - AppView class for ui-app element
- [x] `web/src/app_view.ts` - Uses elementId (getElement via document.getElementById)

### crc-ElementIdVendor.md
**Source Spec:** viewdefs.md (Cross-Cutting: No Direct Element References)
**Implementation:**
- [x] `web/src/element_id_vendor.ts` - Global ElementIdVendor singleton (ensureElementId function)
- [x] `web/src/element_id_vendor.ts` - vendId() returns `ui-{counter}` format

### crc-Widget.md
**Source Spec:** viewdefs.md
**Implementation:**
- [x] `web/src/binding.ts` - Widget class (elementId, variables map, unbindHandlers map)
- [x] `web/src/binding.ts` - Use ElementIdVendor for element ID (ensureElementId)
- [x] `web/src/binding.ts` - Variable-to-Widget relationship via widget property on variable
- [x] `web/src/binding.ts` - hasBinding method for sibling binding lookup
- [x] `web/src/binding.ts` - registerBinding(name, varId, unbindHandler) method
- [x] `web/src/binding.ts` - unbindAll() method (calls all unbind handlers, clears maps)

### crc-BindingEngine.md
**Source Spec:** viewdefs.md, libraries.md
**Implementation:**
- [x] `web/src/binding.ts` - Binding engine with child variable architecture (all bindings create child variables for server-side path resolution)
- [x] `web/src/binding.ts` - Widget management (widgets map keyed by elementId)
- [x] `web/src/binding.ts` - Register unbind handlers with Widget for each binding
- [x] `web/src/binding.ts` - unbindElement calls widget.unbindAll() and removes Widget from map
- [x] `web/src/binding.ts` - store.update sets local value then sends
- [x] `web/src/binding.ts` - Pass widget reference to all bindings
- [x] `web/src/binding.ts` - Duplicate update suppression in VariableStore.update()

### crc-ValueBinding.md
**Source Spec:** viewdefs.md, libraries.md
**Implementation:**
- [x] `web/src/binding.ts` - Value bindings with child variable creation, event selection based on keypress property
- [x] `web/src/binding.ts` - ui-code extended scope (element, value, variable, store)
- [x] `web/src/binding.ts` - Duplicate update suppression in VariableStore.update()
- [x] `web/src/binding.ts` - scrollOnOutput path property for auto-scroll on value update
- [x] `web/src/binding.ts` - Closures use elementId lookup (no direct DOM references)

### crc-EventBinding.md
**Source Spec:** viewdefs.md
**Implementation:**
- [x] `web/src/binding.ts` - Event bindings (combined with BindingEngine)
- [x] `web/src/binding.ts` - Closures use elementId lookup (no direct DOM reference)
- [x] `web/src/binding.ts` - Widget reference for sibling binding access
- [x] `web/src/binding.ts` - syncValueBeforeEvent (check ui-value, compare, sync if changed)
- [x] `web/src/binding.ts` - Keypress modifier support (ctrl, shift, alt, meta)
- [x] `web/src/binding.ts` - Exact modifier matching (matchesModifiers)

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

### crc-FrontendOutgoingBatcher.md
**Source Spec:** protocol.md
**Implementation:**
- [x] `web/src/outgoing_batcher.ts` - Frontend outgoing message batching with 200ms throttle

### crc-LuaSession.md
**Source Spec:** libraries.md, interfaces.md, protocol.md, deployment.md
**Implementation:**
- [x] `internal/lua/runtime.go` - LuaSession struct with per-session Lua VM state
- [x] `internal/lua/runtime.go` - CreateLuaSession, GetLuaSession, ExecuteInSession
- [x] `internal/lua/runtime.go` - HandleFrontendCreate, HandleFrontendUpdate (PathVariableHandler)
- [x] `internal/lua/runtime.go` - AfterBatch for automatic change detection
- [x] `internal/lua/runtime.go` - mutationVersion field for hot-loading
- [x] `internal/lua/runtime.go` - newVersion, getVersion, needsMutation methods
- [x] `internal/server/server.go` - Server.luaSessions map and CreateLuaBackendForSession
- [x] `internal/server/server.go` - Server implements PathVariableHandler interface
- [x] `internal/server/server.go` - luaTrackerAdapter with SetLuaSession/RemoveLuaSession

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

### External: change-tracker package
**Source Spec:** libraries.md, data-models.md
**Package:** `github.com/zot/change-tracker`
**Provides:** Variable management, change detection, object registry, value serialization
**Implementation:**
- [x] External package - see change-tracker repository for details

### crc-FrontendApp.md
**Source Spec:** libraries.md, interfaces.md
**Implementation:**
- [x] `web/src/app.ts` - Frontend app (includes SPA navigation)

### crc-SPANavigator.md
**Source Spec:** libraries.md, interfaces.md
**Implementation:**
- [x] `web/src/app.ts` - SPA navigation (combined with FrontendApp)

### crc-ViewRenderer.md
**Source Spec:** viewdefs.md, libraries.md
**Implementation:**
- [x] `web/src/renderer.ts` - View renderer
- [x] `web/src/renderer.ts` - 3-tier namespace resolution (namespace -> fallbackNamespace -> DEFAULT)
- [x] `web/src/renderer.ts` - Script collection and activation (collectScripts, activateScripts from viewdef.ts)
- [x] `web/src/renderer.ts` - Uses elementId references (no direct DOM storage)

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
- [x] `web/src/binding.ts` - Parse `scrollOnOutput` path property

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
- [x] `internal/config/config.go` - lua.hotload config option

### crc-LuaHotLoader.md
**Source Spec:** deployment.md
**Implementation:**
- [x] `internal/lua/hotloader.go` - LuaHotLoader struct
- [x] `internal/lua/hotloader.go` - Start/Stop file watcher
- [x] `internal/lua/hotloader.go` - Symlink resolution and target watching
- [x] `internal/lua/hotloader.go` - handleFileChange for reload in all sessions
- [x] `internal/server/server.go` - Initialize hot-loader when lua.hotload enabled

### crc-Wrapper.md
**Source Spec:** protocol.md
**Implementation:**
- [x] `internal/lua/wrapper.go` - Wrapper interface and registry
- [x] `internal/lua/viewlist.go` - ViewList wrapper implementation
- [x] `lib/wrapper_example.lua` - Example Lua wrapper (demonstrates custom wrapper pattern)

**Notes:** Wrapper base is implemented in Go. Custom wrappers can be written in Lua following the example pattern.

---

## Historical: Removed Design Elements

The following design elements were removed as their functionality is now provided externally:

- **crc-WatchManager.md** - Merged into crc-LuaBackend.md (per-session watch management)
- **crc-ChangeDetector.md** - Now provided by change-tracker package
- **crc-ObjectRegistry.md** - Now provided by change-tracker package
- **seq-object-registry.md** - Internal to change-tracker package
