# Design: UI Engine

**Source Spec:** specs/main.md, specs/protocol.md, specs/viewdefs.md, specs/deployment.md, specs/interfaces.md, specs/data-models.md, specs/libraries.md, specs/components.md

## Intent

Reactive UI framework with variable-based state synchronization between frontend and backend. Uses a 2-layer server architecture (frontend session management + backend variable management) with embedded Lua for presentation logic. Views are defined via HTML viewdefs with declarative bindings.

## Cross-cutting Concerns

### Hot-Loading Symlink Tracking

**Spec:** main.md "Hot-Loading System"

All hot-loading features (Lua files, viewdefs, etc.) must track symlinks. See spec for full requirements.

**Implementation pattern:**
- `symlinkTargets` map: file path → resolved target directory
- `watchedDirs` map: directory path → reference count
- `scanSymlinks()`: Initial scan for existing symlinks
- `updateSymlinkWatch()`: Add/update watch when symlink changes
- `removeSymlinkWatch()`: Remove watch when symlink deleted
- `resolveReloadPath()`: Map target file changes back to source file

**Components that implement this:**
- LuaHotLoader: `internal/lua/hotloader.go`
- ViewdefHotLoader: `internal/viewdef/hotloader.go`

**Referenced by:** crc-LuaHotLoader.md, crc-ViewdefStore.md, seq-lua-hotload.md, seq-viewdef-hotload.md

## Artifacts

### Variable Protocol System
- [x] crc-Variable.md → `internal/variable/variable.go`, `web/src/variable.ts`
- [x] crc-VariableStore.md → `internal/variable/store.go`, `web/src/connection.ts`
- [x] crc-ProtocolHandler.md → `internal/protocol/handler.go`, `web/src/protocol.ts`
- [x] crc-Wrapper.md → `internal/lua/wrapper.go`, `internal/lua/viewlist.go`
- [x] seq-create-variable.md
- [x] seq-update-variable.md
- [x] seq-watch-variable.md
- [x] seq-destroy-variable.md
- [x] seq-wrapper-transform.md

### Presenter System
- [x] crc-Presenter.md → `internal/presenter/presenter.go`, `lib/presenter.lua`
- [x] crc-AppPresenter.md → `lib/app.lua`
- [x] crc-ListPresenter.md → `lib/list.lua`
- [x] seq-create-presenter.md
- [x] seq-navigate-page.md

### Viewdef System
- [x] crc-Viewdef.md → `internal/viewdef/viewdef.go`, `web/src/viewdef.ts`
- [x] crc-ViewdefStore.md → `internal/viewdef/store.go`, `internal/viewdef/hotloader.go`, `web/src/viewdef_store.ts` *(hot-reload)*
- [x] crc-View.md → `web/src/view.ts` *(ui-viewdef, rerender)*
- [x] crc-ViewList.md → `web/src/viewlist.ts`, `internal/lua/viewlist.go`
- [x] crc-ViewListItem.md → `internal/lua/viewlistitem.go`
- [x] crc-AppView.md → `web/src/app_view.ts`
- [x] crc-Widget.md → `web/src/binding.ts`
- [x] crc-BindingEngine.md → `web/src/binding.ts`
- [x] crc-ValueBinding.md → `web/src/binding.ts`
- [x] crc-EventBinding.md → `web/src/binding.ts`
- [x] crc-HtmlBinding.md → `web/src/binding.ts`
- [x] seq-load-viewdefs.md
- [x] seq-viewdef-delivery.md
- [x] seq-render-view.md *(set ui-viewdef)*
- [x] seq-viewlist-update.md
- [x] seq-viewlist-presenter-sync.md
- [x] seq-bind-element.md *(pass viewElementId)*
- [x] seq-handle-event.md
- [x] seq-handle-keypress-event.md
- [x] seq-input-value-binding.md
- [x] seq-viewdef-hotload.md

### Session System
- [x] crc-Session.md → `internal/session/session.go`
- [x] crc-SessionManager.md → `internal/session/manager.go`
- [x] crc-Router.md → `internal/router/router.go`, `web/src/router.ts`
- [x] seq-create-session.md
- [x] seq-session-create-backend.md
- [x] seq-activate-tab.md
- [x] seq-navigate-url.md
- [x] seq-frontend-reconnect.md

### Backend System
- [x] crc-Backend.md → `internal/backend/backend.go`
- [x] crc-LuaBackend.md → `internal/backend/lua.go`
- [x] seq-backend-watch.md
- [x] seq-backend-detect-changes.md

### Communication System
- [x] crc-WebSocketEndpoint.md → `internal/server/websocket.go`, `web/src/connection.ts`
- [x] crc-HTTPEndpoint.md → `internal/server/http.go`
- [x] crc-SharedWorker.md → `web/src/worker.ts`
- [x] crc-MessageRelay.md → `internal/server/relay.go`
- [x] crc-MessageBatcher.md → `internal/protocol/batcher.go`, `web/src/batcher.ts`
- [x] crc-FrontendOutgoingBatcher.md → `web/src/outgoing_batcher.ts`
- [x] seq-frontend-connect.md
- [x] seq-backend-connect.md
- [x] seq-relay-message.md
- [x] seq-frontend-outgoing-batch.md

### Backend Socket System
- [x] crc-BackendSocket.md → `internal/server/backend_socket.go`
- [x] crc-PendingResponseQueue.md → `internal/server/pending.go`
- [x] seq-backend-socket-accept.md
- [x] seq-poll-pending.md

### Lua Runtime System
- [x] crc-LuaSession.md → `internal/lua/runtime.go`, `internal/server/server.go`
- [x] crc-LuaResolver.md → `internal/lua/resolver.go` *(implements change-tracker.Resolver)*
- [x] crc-LuaVariable.md → `internal/lua/runtime.go`
- [x] crc-LuaPresenterLogic.md → `lib/presenter_logic.lua`
- [x] crc-LuaHotLoader.md → `internal/lua/hotloader.go`
- [x] seq-lua-executor-init.md
- [x] seq-lua-session-init.md
- [x] seq-lua-execute.md
- [x] seq-load-lua-code.md
- [x] seq-lua-handle-action.md
- [x] seq-lua-hotload.md
- [x] seq-prototype-mutation.md

### Backend Library System
- [x] crc-PathNavigator.md → `lib/go/path.go`, `lib/lua/path.lua`, `web/src/path.ts`
- [x] crc-BackendConnection.md → `lib/go/connection.go`
- [x] seq-path-resolve.md
- [x] seq-backend-refresh.md

### Frontend Library System
- [x] crc-FrontendApp.md → `web/src/app.ts`
- [x] crc-SPANavigator.md → `web/src/app.ts`
- [x] crc-ViewRenderer.md → `web/src/renderer.ts`
- [x] crc-WidgetBinder.md → `web/src/binding.ts`
- [x] seq-spa-navigate.md
- [x] seq-render-view.md
- [x] ui-app-shell.md

### Bundle System
- [x] crc-Bundle.md → `internal/bundle/bundle.go`, `internal/bundle/bundle_test.go`, `cli/commands.go`

### Cross-Cutting
- [x] crc-Config.md → `internal/config/config.go`
- [x] crc-ElementIdVendor.md → `web/src/element_id_vendor.ts`
- [x] crc-ObjectReference.md → `internal/variable/variable.go`, `web/src/variable.ts`
- [x] crc-PathSyntax.md → `internal/path/syntax.go`, `web/src/binding.ts`
- [x] manifest-ui.md
- [x] seq-server-startup.md
- [x] seq-bootstrap.md
- [x] seq-app-startup.md

### Test Designs
- [x] test-HotLoader.md
- [x] test-Lua.md

## Gaps

### Incomplete Implementation

None identified.

### Spec → Design Gaps

None identified.

### Design → Code Gaps

- [x] D1: Method path access validation incomplete
  - ~~**Spec**: viewdefs.md defines `method()` with `access=rw` for read/write methods (Lua only)~~
  - ~~**Code**: `web/src/binding.ts:559` rejected `access=rw` for method paths~~
  - Fixed: Now allows `r`, `action`, or `rw` for `method()` paths

### Oversights

- [x] O1: Consolidate baseDir across components
  - ~~Currently derived as `filepath.Dir(luaDir)` in both LuaSession and HotLoader~~
  - Now uses `config.Server.Dir` as single source of truth
- [ ] O2: Frontend test coverage
  - No TypeScript unit tests (`*.test.ts`) for web/src/ components
  - Test designs exist (test-Frontend.md) but no implementation
- [ ] O3: ChanSvc utility undocumented
  - `internal/server/svc.go` provides channel-based service pattern
  - Used for concurrency but has no CRC card (minor - utility code)
