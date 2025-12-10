# Architecture

**Entry point to the design - shows how design elements are organized into logical systems**

**Sources**: main.md, protocol.md, viewdefs.md, deployment.md, interfaces.md, data-models.md, libraries.md, components.md

---

## Systems

### Variable Protocol System

**Purpose**: Core protocol for variable identity, values, properties, and message handling

**Design Elements**:
- crc-Variable.md
- crc-VariableStore.md
- crc-ProtocolHandler.md
- crc-WatchManager.md
- seq-create-variable.md
- seq-update-variable.md
- seq-watch-variable.md
- seq-destroy-variable.md

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
- crc-BindingEngine.md
- crc-ValueBinding.md
- crc-EventBinding.md
- seq-load-viewdefs.md
- seq-bind-element.md
- seq-handle-event.md

### Session System

**Purpose**: Session management, URL routing, tab coordination, and connection timeout handling

**Design Elements**:
- crc-Session.md
- crc-SessionManager.md
- crc-Router.md
- seq-create-session.md
- seq-activate-tab.md
- seq-navigate-url.md
- seq-frontend-reconnect.md

### Communication System

**Purpose**: WebSocket/HTTP transport, SharedWorker coordination, message relay

**Design Elements**:
- crc-WebSocketEndpoint.md
- crc-HTTPEndpoint.md
- crc-SharedWorker.md
- crc-MessageRelay.md
- seq-frontend-connect.md
- seq-backend-connect.md
- seq-relay-message.md

### Backend Socket System

**Purpose**: Local socket API for backend programs with multi-protocol support

**Design Elements**:
- crc-BackendSocket.md
- crc-ProtocolDetector.md
- crc-PacketProtocol.md
- crc-PendingResponseQueue.md
- seq-backend-socket-accept.md
- seq-poll-pending.md

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

**Purpose**: Embedded Lua backend for presentation logic

**Design Elements**:
- crc-LuaRuntime.md
- crc-LuaPresenterLogic.md
- seq-load-lua-code.md
- seq-lua-handle-action.md

### Backend Library System

**Purpose**: Go/Lua library for backend integration (connection, path navigation, change detection)

**Design Elements**:
- crc-BackendConnection.md
- crc-PathNavigator.md
- crc-ChangeDetector.md
- seq-backend-refresh.md
- seq-path-resolve.md

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
