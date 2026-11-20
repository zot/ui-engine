# Design: UI Engine

**Source Spec:** specs/main.md, specs/protocol.md, specs/viewdefs.md, specs/deployment.md, specs/interfaces.md, specs/data-models.md, specs/libraries.md, specs/components.md, specs/js-api.md

## Intent

Reactive UI framework with variable-based state synchronization between frontend and backend. Uses a 2-layer server architecture (frontend session management + backend variable management) with embedded Lua for presentation logic. Views are defined via HTML viewdefs with declarative bindings.

## Cross-cutting Concerns

### Hot-Loading Through Activation Entries

**Spec:** main.md "Hot-Loading System", app-discovery.md

Hot loaders watch app directories *through* their `lua/<app>` entry (a
symlink or a Windows junction), never through the resolved target. File-system
events then arrive under the same path the file was loaded from, so no
target-to-link reverse mapping is kept.

**Implementation pattern:**
- AppDiscovery (`internal/site/discovery.go`) lists the active apps; each
  `App.Dir` is the entry path `<site>/lua/<app>`
- Both hot loaders watch `lua/` itself and re-run discovery when an entry is
  created or removed, adding and dropping per-app watches
- The Lua hot loader watches every directory under each App.Dir (dot-dirs
  skipped); the viewdef hot loader watches each source's `viewdefs/`
- Lua reloads are debounced per tracking key, so one file seen through two
  watched paths (the entry and `apps/`) reloads once

**Components that implement this:**
- LuaHotLoader: `internal/lua/hotloader.go`
- ViewdefHotLoader: `internal/viewdef/hotloader.go`

**Referenced by:** crc-LuaHotLoader.md, crc-ViewdefStore.md, seq-lua-hotload.md, seq-viewdef-hotload.md, seq-app-discovery.md

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
- [x] crc-AppDiscovery.md → `internal/site/discovery.go`
- [x] crc-ViewdefManager.md → `internal/viewdef/viewdefs.go`, `cli/exports.go`
- [x] crc-Server.md → `internal/server/server.go`, `internal/server/dangerous.go`
- [x] seq-app-discovery.md → `internal/site/discovery.go`, `internal/viewdef/viewdefs.go`, `internal/viewdef/hotloader.go`, `internal/lua/hotloader.go`, `internal/server/server.go`
- [x] crc-ViewdefStore.md → `internal/viewdef/store.go`, `internal/viewdef/hotloader.go`, `web/src/viewdef_store.ts` *(hot-reload)*
- [x] crc-View.md → `web/src/view.ts`, `web/src/namespace.ts`
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
- [x] seq-no-flash.md *(no-flash view rendering)*

### Session System
- [x] crc-Session.md → `internal/server/session.go`
- [x] crc-SessionRef.md → `internal/server/sessionref.go`, `internal/server/tailor.go`, `cli/exports.go`
- [x] seq-session-ref.md → `internal/server/sessionref.go`, `internal/server/server.go`, `internal/server/websocket.go`
- [x] crc-Tailor.md → `internal/server/tailor.go`
- [x] seq-tailor.md → `internal/server/tailor.go`, `internal/server/server.go`, `internal/server/session.go`
- [x] crc-ChanSvc.md → `internal/server/svc.go`
- [x] crc-HostOp.md → `internal/server/hostop.go`, `cli/exports.go`
- [x] seq-host-op.md → `internal/server/hostop.go`
- [x] seq-session-executor.md → `internal/server/session.go`, `internal/server/session_manager.go`, `internal/server/websocket.go`, `internal/server/svc.go`
- [x] crc-SessionManager.md → `internal/server/session_manager.go`
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
- [x] crc-ServerOutgoingBatcher.md → `internal/server/outgoing_batcher.go`
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
- [x] crc-LuaTrackerAdapter.md → `internal/server/server.go`
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
- [x] crc-Module.md → `internal/lua/module.go`
- [x] crc-PortableOS.md → `internal/lua/portable.go`, `internal/lua/runtime.go`
- [x] crc-ExecAsync.md → `internal/server/execasync.go`, `internal/server/server.go`
- [x] seq-unload-module.md → `internal/lua/runtime.go`, `internal/lua/hotloader.go`
- [x] seq-require-lua-file.md → `internal/lua/runtime.go`
- [x] seq-session-timer.md → `internal/lua/runtime.go`, `internal/server/server.go`

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

### Variable Browser
- [x] crc-VariableBrowser.md → `internal/server/variables_html.go`
- [x] crc-HTTPEndpoint.md → `internal/server/http.go`

### Test Designs
- [x] test-AppDiscovery.md → `internal/lua/collisions_test.go`, `internal/site/discovery_test.go`, `internal/viewdef/viewdefs_test.go`, `internal/server/site_test.go`, `internal/server/appstatic_test.go`, `internal/lua/appwatch_test.go`, `internal/viewdef/hotloader_test.go`
- [x] test-HotLoader.md
- [x] test-ArrayGetter.md → `internal/lua/arrayget_test.go`
- [x] test-Lua.md
- [x] test-SessionTimer.md → `internal/lua/shutdown_race_test.go`
- [x] test-PortableOS.md → `internal/lua/portable_test.go`
- [x] test-ExecAsync.md → `internal/server/execasync_test.go`, `internal/lua/portable_test.go`
- [x] test-SessionRef.md → `internal/server/sessionref_test.go`, `internal/server/dangerous_test.go`, `internal/server/session_lua_test.go`
- [x] test-LuaTrackerAdapter.md → `internal/server/store_adapter_test.go`
- [x] test-WebSocketEndpoint.md → `internal/server/connection_test.go`
- [x] test-Tailor.md → `internal/server/tailor_test.go`
- [x] test-ChanSvc.md → `internal/server/svc_test.go`
- [x] test-OneExecutor.md → `internal/server/one_executor_test.go`, `internal/lua/hotloader_test.go`, `internal/lua/shutdown_race_test.go`
- [x] test-Bundle.md → `internal/bundle/bundle_test.go`
- [x] test-HostOp.md → `internal/server/hostop_test.go`
- [x] test-SessionExecutor.md → `internal/server/session_executor_test.go`

## Gaps

- T1: R13 retired by R135 (2026-10-07 per-app hot-loading: symlinkTargets retired; app dirs are watched through their lua/<app> entry)
- T2: R20 retired by R136 (2026-10-07 per-app hot-loading: symlinkTargets retired; app dirs are watched through their lua/<app> entry)
- T3: R166 retired by R173 (2026-11-18 dangerous execution: ExecuteInSession now delegates to PrimitiveExecuteInSession)
- T4: R203 retired by R218 (stop became final and non-blocking (R218-R223))
- T5: R204 retired by R224 (the executor belongs to the session (R224-R228))
- T6: R155 retired by R238 (nothing restarts a Lua session; the real case is vended-ID reuse)
- T7: R173 retired by R229 (the Lua session moved into the session and ExecuteInSessionAsync became PostUpdate (R229-R237))
- T8: R116 retired by R262 (Item 14: there is no separate Lua executor to dequeue from)
- T9: R115 retired by R260 (Item 14: one executor per session; shutdown is executor work (R260), with a bounded wait from outside (R261), not a join of a Lua goroutine)
- T10: R172 retired by R265 (Item 12c: every caller in ui-engine, Frictionless and ark was moved, so the default flipped to not set)
- T11: R150 retired by R266 (Item 11c: ui.exec removed once ark moved off it; ui.execAsync is the one way to run a program)
- T12: R251 retired by R266 (Item 11c: restated without reference to the removed ui.exec)
- T13: R253 retired by R267 (Item 11c: the deprecation ended in removal)

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
- [x] O3: ChanSvc utility undocumented
  - `internal/server/svc.go` provides channel-based service pattern
  - Used for concurrency but has no CRC card (minor - utility code)

- [ ] O4: R118 (Server.Shutdown lock-release before per-session Shutdown) has no automated regression test; a deterministic deadlock test needs a full Server harness (config, wsEndpoint, SessionManager, a re-entrant deferred task). Enforced by code review plus TestShutdownJoinsInFlightExecutor proving the join blocks; verified manually via ark stop.- [ ] O5: Legacy design debt predating minispec 4 (found 2026-10-07 at init): 38 CRC cards lack a **Requirements:** field, 10 design files are unlisted in Artifacts, crc-Bundle/crc-LuaResolver name missing sequences, and several seqs reuse step IDs. phase design/validate stay red until it is paid; app-discovery work fixed only the cards it touched.
- [x] O6: Windows junction hot-loading unverified (resolved 2026-10-07: test-AppDiscovery runs on the Windows VM with junction app entries; it caught and fixed a junction tracking-key split in queueReload) (test-AppDiscovery.md, manual case): fsnotify reporting events under a `lua/<app>` junction path was measured only on Linux (symlinks, 2026-10-07). Needs a run on the Windows test VM from a scheduled task, since ssh's RedirectionTrust refuses non-admin junctions.
- [ ] O7: inotify gives two paths naming one directory a single watch, so when an app living under `apps/` is deactivated, removing its `lua/<app>` watches also drops the `apps/<app>` watch the Lua hot loader set at start; that app's `init.lua` then stops hot-reloading until restart. Low impact (the app is inactive); fix by reference-counting watches by resolved directory if it matters.
- [ ] O8: A failed `CreateLuaSession` (main.lua raising) never shuts down the Lua session it made: `CreateLuaBackendForSession` only drops it, so its VM and executor goroutine leak, and until `SessionManager` forgets the session it is still attached and can be found. Found 2026-11-19 designing the Lua session's move into Session (Item 8); predates it. Needs a spec line before a fix.
- [ ] O9: Creating a non-root server variable through the variable store panics: the adapter passes the Lua object as the value of `tracker.CreateVariableWithId`, and the tracker panics for any child with a value ("cannot provide both parentID and value"). A nil `*LTable` is a non-nil interface, so even no object counts. Reached by Lua `session:createVariable(parent, obj)`. Found 2026-11-19 writing Item 9a's tests; the pre-9a adapter made the same call. Needs a spec line for what a server child variable is.
- [ ] O10: `lib/lua/session.lua` is never loaded by a real session: `loadSessionModule` runs `require("session")`, which ui-engine's own resolver does not find even with the file in the site's `lua` directory (measured 2026-11-19: `_SessionModule` and `package.path` both nil), so sessions use the inline fallback table. Its `Variable:getValue()` and `getProperty()`, and the Go get-value and get-property hooks that only install into that module, are unreachable. Frictionless ships no copy. Decide whether to load it or delete it.
- [ ] O11: Registering presenters at URL paths after the session ID (`interfaces.md`, SPA History Management) is specified but not built. Nothing lets a backend register a path, and `http.go` serves the SPA for every path under a session ("it will handle the routing client-side"). Three unconnected pieces exist: `SessionManager.urlPaths` (deleted 2026-11-19, Item 9c), `internal/router` (imported by nothing; `crc-Router.md`, `seq-navigate-url.md`), and `web/src/router.ts` (its `register` is never called). Building it decides where routes live; the server comment points at the browser.
- [ ] O12: A dangerous `ExecuteInSession` called directly by a `go` statement is reported at the wrong site: the site finder takes the first caller frame, which is the runtime's goroutine entry (`runtime/asm_amd64.s`), not the line with the `go` statement. The panic still fires; only the message misleads. Found 2026-11-19 in Item 12c's trial flip, from `go s.ExecuteInSession(...)` in a test. Walking past runtime frames, or naming the goroutine's creator, would fix it.
