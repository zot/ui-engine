# Architecture

**Entry point to the design - shows how design elements are organized into logical systems**

**Sources**: main.md, protocol.md, viewdefs.md, deployment.md, interfaces.md, data-models.md, libraries.md, components.md

---

## Server Layers

The UI server has two conceptual layers (see main.md "UI Server Architecture"):

### Frontend Layer (Session Management)

Manages browser connections and sessions. Lightweight - routes messages to backends.

**Components**: Session, SessionManager, WebSocketEndpoint, HTTPEndpoint

### Backend Layer (Variable Management)

Each session has an associated Backend that handles variable management. Two implementations:

- **LuaBackend**: Hosted Lua with per-session change-tracker (processes messages locally)
- **ProxiedBackend**: Relays messages to external backend (future)

**Components**: Backend (interface), LuaBackend

---

## Systems

### Variable Protocol System

**Purpose**: Core protocol for variable identity, values, properties, wrappers, and message handling

**Design Elements**:
- crc-Variable.md
- crc-VariableStore.md
- crc-ProtocolHandler.md
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
- crc-ViewListItem.md
- crc-AppView.md
- crc-Widget.md
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
- seq-handle-keypress-event.md
- seq-input-value-binding.md

### Session System (Frontend Layer)

**Purpose**: Session management, URL routing, and tab coordination

**Design Elements**:
- crc-Session.md
- crc-SessionManager.md
- crc-Router.md
- seq-create-session.md
- seq-session-create-backend.md
- seq-activate-tab.md
- seq-navigate-url.md
- seq-frontend-reconnect.md

**Notes**:
- Session is lightweight frontend layer component
- Delegates all protocol messages to Backend
- Backend field holds LuaBackend or ProxiedBackend instance

### Backend System

**Purpose**: Per-session backend implementations for variable management and change detection

**Design Elements**:
- crc-Backend.md
- crc-LuaBackend.md
- seq-backend-watch.md
- seq-backend-detect-changes.md

**Notes**:
- Backend is an interface; LuaBackend is the hosted implementation
- Each LuaBackend owns its own change-tracker.Tracker (per-session, not global)
- Watch tallies and watcher maps are per-session, fixing the global map collision bug
- ProxiedBackend (future) will relay messages to external backend

### Communication System

**Purpose**: WebSocket/HTTP transport, SharedWorker coordination, message relay and batching

**Design Elements**:
- crc-WebSocketEndpoint.md
- crc-HTTPEndpoint.md
- crc-SharedWorker.md
- crc-MessageRelay.md
- crc-MessageBatcher.md
- crc-FrontendOutgoingBatcher.md
- seq-frontend-connect.md
- seq-backend-connect.md
- seq-relay-message.md
- seq-frontend-outgoing-batch.md

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

### Lua Runtime System

**Purpose**: Embedded Lua backend for presentation logic with per-session isolation

**Design Elements**:
- crc-LuaSession.md (per-session Lua environment with isolated VM)
- crc-LuaVariable.md
- crc-LuaPresenterLogic.md
- crc-LuaHotLoader.md (file watcher for hot-loading)
- seq-lua-executor-init.md
- seq-lua-session-init.md
- seq-lua-execute.md
- seq-load-lua-code.md
- seq-lua-handle-action.md
- seq-lua-hotload.md (hot-loading flow with prototype management)
- seq-prototype-mutation.md (post-load mutation processing)

**Notes**:
- **Per-Session Isolation**: Server owns `luaSessions map[string]*LuaSession`
- Each LuaSession has its own Lua VM state (complete isolation between sessions)
- Server implements PathVariableHandler, routing to per-session LuaSession
- luaTrackerAdapter coordinates variable operations across per-session backends
- **Hot-Loading**: When `--hotload` enabled, LuaHotLoader watches lua directory
- Modified files are re-executed in all active sessions
- Symlink targets are also watched for changes
- **Prototype Management**: `session:prototype()` and `session:create()` enable automatic instance tracking and schema migration during hot-reload

### Backend Library System

**Purpose**: Path navigation, change detection, and object identity for backend integration

**External Package**: Core tracking provided by `change-tracker` (`github.com/zot/change-tracker`)
- Variable management, change detection, object registry, value serialization
- See change-tracker docs for details

**Design Elements** (UI-specific extensions):
- crc-PathNavigator.md
- crc-BackendConnection.md
- seq-path-resolve.md
- seq-backend-refresh.md

**Notes**:
- BackendConnection used by external Go backends (connected backend mode)
- Embedded Lua uses LuaSession instead of BackendConnection

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
- crc-ElementIdVendor.md (global element ID vendor - no direct DOM references)
- crc-ObjectReference.md
- crc-PathSyntax.md
- manifest-ui.md
- seq-server-startup.md
- seq-bootstrap.md
- seq-app-startup.md

---

*This file serves as the architectural "main program" - start here to understand the design structure*
