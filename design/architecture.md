# Architecture

**Entry point to the design - shows how design elements are organized into logical systems**

**Sources**: main.md, protocol.md, viewdefs.md, deployment.md, interfaces.md, data-models.md, libraries.md, components.md

---

## Systems

### Variable Protocol System

**Purpose**: Core protocol for variable identity, values, properties, wrappers, and message handling

**Design Elements**:
- crc-Variable.md
- crc-VariableStore.md
- crc-ProtocolHandler.md
- crc-WatchManager.md
- crc-Wrapper.md
- seq-create-variable.md
- seq-update-variable.md
- seq-watch-variable.md
- seq-destroy-variable.md
- seq-wrapper-transform.md

### Presenter System

**Purpose**: Manage presenter types, instances, and standard presenters (App, List)

**Design Elements**:
- crc-Presenter.md
- crc-AppPresenter.md
- crc-ListPresenter.md
- seq-create-presenter.md
- seq-navigate-page.md

### Viewdef System

**Purpose**: View definitions, binding engine, and UI rendering

**Design Elements**:
- crc-Viewdef.md
- crc-ViewdefStore.md
- crc-View.md
- crc-ViewList.md
- crc-ViewItem.md
- crc-AppView.md
- crc-BindingEngine.md
- crc-ValueBinding.md
- crc-EventBinding.md
- seq-load-viewdefs.md
- seq-viewdef-delivery.md
- seq-render-view.md
- seq-viewlist-update.md
- seq-viewlist-presenter-sync.md
- seq-bind-element.md
- seq-handle-event.md

### Session System

**Purpose**: Session management, URL routing, and tab coordination

**Design Elements**:
- crc-Session.md
- crc-SessionManager.md
- crc-Router.md
- seq-create-session.md
- seq-activate-tab.md
- seq-navigate-url.md
- seq-frontend-reconnect.md

### Communication System

**Purpose**: WebSocket/HTTP transport, SharedWorker coordination, message relay and batching

**Design Elements**:
- crc-WebSocketEndpoint.md
- crc-HTTPEndpoint.md
- crc-SharedWorker.md
- crc-MessageRelay.md
- crc-MessageBatcher.md
- seq-frontend-connect.md
- seq-backend-connect.md
- seq-relay-message.md

### Backend Socket System

**Purpose**: Local socket API for external backends (Go, etc.) with session-wrapped protocol

**Design Elements**:
- crc-BackendSocket.md
- crc-PendingResponseQueue.md
- seq-backend-socket-accept.md
- seq-poll-pending.md

**Notes**:
- Supports connected backend only, or hybrid mode with embedded Lua
- Session-wrapped batching: `{"session": "abc123", "messages": [...]}`
- Backend creates variable 1 (unless hybrid mode with Lua creating it)

### Storage System

**Purpose**: Persistent and in-memory storage for variables and objects

**Design Elements**:
- crc-StorageBackend.md
- crc-MemoryStorage.md
- crc-SQLiteStorage.md
- crc-PostgresStorage.md
- seq-store-variable.md
- seq-retrieve-variable.md

### MCP Integration System

**Purpose**: AI assistant integration via Model Context Protocol

**Design Elements**:
- crc-MCPServer.md
- crc-MCPResource.md
- crc-MCPTool.md
- seq-mcp-create-session.md
- seq-mcp-create-presenter.md
- seq-mcp-receive-event.md

### Lua Runtime System

**Purpose**: Embedded Lua backend for presentation logic with session-based architecture

**Design Elements**:
- crc-LuaRuntime.md
- crc-LuaSession.md
- crc-LuaVariable.md
- crc-LuaPresenterLogic.md
- seq-lua-executor-init.md
- seq-lua-session-init.md
- seq-lua-execute.md
- seq-load-lua-code.md
- seq-lua-handle-action.md

### Backend Library System

**Purpose**: Path navigation, change detection, and object identity for backend integration

**Design Elements**:
- crc-PathNavigator.md
- crc-ChangeDetector.md
- crc-ObjectRegistry.md
- crc-BackendConnection.md
- seq-path-resolve.md
- seq-backend-refresh.md
- seq-object-registry.md

**Notes**:
- BackendConnection used by external Go backends (connected backend mode)
- Embedded Lua uses LuaSession instead of BackendConnection
- ObjectRegistry provides identity-based serialization for Go backends (requires Go 1.25+)

### Frontend Library System

**Purpose**: Browser-side SPA navigation, view rendering, widget bindings

**Design Elements**:
- crc-FrontendApp.md
- crc-SPANavigator.md
- crc-ViewRenderer.md
- crc-WidgetBinder.md
- seq-spa-navigate.md
- seq-render-view.md
- ui-app-shell.md

---

## Cross-Cutting Concerns

**Design elements that span multiple systems**

**Design Elements**:
- crc-Config.md
- crc-ObjectReference.md
- crc-PathSyntax.md
- manifest-ui.md
- seq-server-startup.md
- seq-bootstrap.md
- seq-app-startup.md

---

*This file serves as the architectural "main program" - start here to understand the design structure*
