# Requirements

## Feature: Remove Prototype
**Source:** specs/remove-prototype.md

- **R1:** LuaSession must have a RemovePrototype(name string, children bool) method
- **R2:** RemovePrototype removes the named prototype from prototypeRegistry
- **R3:** RemovePrototype removes the corresponding entry from instanceRegistry
- **R4:** When children=true, RemovePrototype removes all prototypes whose name starts with "NAME." (dot-separated children)
- **R5:** When children=false, RemovePrototype removes only the exact named prototype
- **R6:** RemovePrototype returns silently if the prototype doesn't exist (no error)
- **R7:** Existing instances retain their metatables after prototype removal (no instance destruction)

## Feature: Module Tracking and Unloading
**Source:** specs/module-tracking.md

- **R8:** A Module struct must track prototypes, presenterTypes, and wrappers registered by a single module
- **R9:** LuaSession must have a moduleDirectories field mapping directory paths to their modules
- **R10:** LuaSession must have an unloadDirectory(name) method exposed to Lua as session:unloadDirectory(name)
- **R11:** unloadDirectory must call unloadModule for each module in the directory
- **R12:** unloadDirectory must remove the directory entry from moduleDirectories after unloading modules
- **~~R13:~~** (Retired T1 — see R135) unloadDirectory must clean up HotLoader state: watchers, symlinkTargets, pendingReloads for the directory
- **R14:** LuaSession must have an unloadModule(moduleName) method exposed to Lua as session:unloadModule(name)
- **R15:** unloadModule must remove the module's entry from loadedModules
- **R16:** unloadModule must remove prototypes registered by the module from prototypeRegistry (using RemovePrototype)
- **R17:** unloadModule must remove presenter types registered by the module from presenterTypes
- **R18:** unloadModule must remove wrappers registered by the module from wrapperRegistry
- **R19:** unloadModule must remove viewdefs registered by the module from viewdefManager
- **~~R20:~~** (Retired T2 — see R136) unloadModule must clean up HotLoader state for the module file: watches, symlinkTargets, pendingReloads
- **R21:** (inferred) Module registration must happen during require() or RequireLuaFile() calls to track resources
- **R22:** (inferred) A directory can contain multiple modules

## Feature: Debug Error Display
**Source:** specs/debug-error-display.md

- **R23:** Variables with errors must display error text in the debug tree view
- **R24:** Error text must be visually distinct using red color styling
- **R25:** Error display must not interfere with other variable information (ID, type, path, value)

## Feature: JavaScript API
**Source:** specs/js-api.md

- **R26:** UIApp instance must be exposed as `window.uiApp` after initialization
- **R27:** UIApp must provide an `updateValue(elementId, value?)` method
- **R28:** updateValue must look up the widget for the given elementId
- **R29:** updateValue must get the variable ID for the `ui-value` binding from the widget
- **R30:** updateValue must send an update with the provided value, or the element's `value` property if value is undefined
- **R31:** updateValue must be a no-op if the element has no `ui-value` binding

## Feature: No-Flash View Rendering
**Source:** specs/no-flash.md

- **R32:** View render must use ancestor-aware timer buffering to prevent visual flashing
- **R33:** New view elements must be created with `.ui-new-view` class (hidden by CSS)
- **R34:** On re-render, old elements must get `.ui-obsolete-view` class instead of immediate removal
- **R35:** After 100ms timer, obsolete elements must be removed and new elements revealed
- **R36:** Views inside an ancestor's buffer (detected via `closest('.ui-new-view')`) must render normally
- **R37:** Each view must have a unique viewClass (e.g., "ui-view-42") to identify its elements
- **R38:** View destruction must clear any pending buffer timeout
- **R50:** On re-render, obsolete elements must have their `id` attribute removed to prevent `getElementById` from finding stale elements
- **R51:** View must filter internal CSS classes (`ui-view-*`, `ui-new-view`, `ui-obsolete-view`) from the captured `originalClass` to prevent class leakage between views
- **R52:** View must apply originalClass and originalStyle to the first rendered element that is not a `<script>` or `<style>` element

## Feature: Server-to-Frontend Batching
**Source:** specs/protocol.md

- **R39:** Server must send batched messages to frontend as JSON arrays
- **R40:** Frontend must detect and handle incoming JSON array batches
- **R41:** Frontend must start outgoing batch timer BEFORE processing incoming messages
- **R42:** ServerOutgoingBatcher must group messages by connection and send one batch per connection

## Feature: Frontend-Vended Variable IDs
**Source:** specs/protocol.md

- **R43:** Frontend must vend its own variable IDs starting from 2 (incrementing)
- **R44:** Server root variable (app variable) must use ID 1
- **R45:** Server-created variables (other than root) must use negative IDs starting from -1 (decrementing)
- **R46:** Create message must include the `id` field with the sender-vended ID
- **R47:** (inferred) No createResponse message is needed - create is push-only
- **R48:** (inferred) VariableStore.create() must be synchronous (no Promise)
- **R49:** Connection class must maintain nextVarId counter for frontend ID vending

## Feature: Empty Table As Empty Array
**Source:** specs/viewdefs.md

- **R152:** Reading a Lua value as an array (`ArrayGetter`, behind ViewList sync and `ArrayGet`) treats an untyped table with no keys other than `_`-prefixed ones as an empty array, without error. A table with non-array keys or a type is still an error. The general array test (`isArray`), which decides whether a table is serialized as an array or an object, is unchanged, so an empty table still serializes as an object.

## Feature: Null View Clearing
**Source:** specs/viewdefs.md

- **R53:** When a rendered view's `type` property becomes empty, the view must clear its rendered content from the DOM
- **R54:** After clearing, a placeholder element must be inserted preserving the view's element ID so it can re-render later
- **R55:** After clearing, the view must be added to the pending views list
- **R56:** (inferred) Clearing must destroy child views and viewlists to prevent resource leaks

## Feature: Variable Browser
**Source:** specs/variable-browser.md

- **R57:** Backend must serve a JSON endpoint at `/{session-id}/variables.json` returning the variable array
- **R58:** Backend must serve a static HTML page at `/{session-id}/variables` (replacing server-rendered HTML)
- **R59:** DebugVariable must include `computeTime` and `maxComputeTime` fields (formatted duration strings)
- **R60:** DebugVariable must include `active` (bool), `access` (string), and `diags` (string array) fields
- **R61:** DebugVariable must include `depth` (int) for tree indentation
- **R62:** JSON endpoint must accept `?diag=N` query parameter to set tracker DiagLevel before collecting
- **R63:** Browser must display variables in an HTML table with fixed header; columns content-sized and left-justified with a spacer column extending the table to full container width
- **R64:** Browser must support tree mode with indented rows and expand/collapse per node; clicking the path name must also toggle expand/collapse
- **R65:** Browser must support flat mode (the default) with all variables as flat sortable rows
- **R66:** Flat/tree toggle must switch between display modes; flat is listed first and is the default
- **R67:** Refresh button must reload data manually (default interaction)
- **R68:** Poll toggle must enable automatic polling at selectable intervals (1s, 2s, 5s)
- **R69:** Column picker must allow showing/hiding individual columns
- **R70:** Default visible columns: Diags, ID, Path, Type, Value, Time, Error
- **R71:** Default hidden columns: GoType, Max Time, Access, Active, Props
- **R72:** In flat mode, column headers must be clickable to sort ascending/descending; numeric columns (Time, Max Time) must default to descending on first click
- **R73:** Diagnostics toggle button must appear per-row when `diags` is non-empty
- **R74:** Clicking the diag toggle must expand/collapse a sub-row showing diagnostic messages
- **R75:** Value cells must show truncated text with a tooltip containing the full JSON
- **R76:** Error cells must use red highlight styling
- **R77:** (inferred) The HTML page must use no external dependencies or build step
- **R78:** (inferred) ID column must be fixed-width to prevent Path column from shifting when data changes
- **R79:** (inferred) Column display order must match spec table: Diags, ID, Path, Type, GoType, Value, Changes, Time, Avg Time, Max Time, Error, Access, Active, Props

## Feature: Change Count Tracking
**Source:** specs/change-count.md

- **R80:** DebugVariable must include a `changeCount` field with the variable's change count
- **R81:** HandleVariablesJSON must set an `X-Change-Count` response header with the tracker's global change count
- **R82:** Variable browser must display a Changes column (default hidden) showing the variable's change count
- **R83:** Variable browser must display an Avg Time column (default hidden) computed as ComputeTime / tracker refresh count (from X-Change-Count header)

## Feature: elementId Variable Property
**Source:** specs/element-id-property.md

- **R84:** Every variable created with a widget reference must have `elementId` set in its properties
- **R85:** `elementId` must be set centrally in `VariableStore.create()` when a `widget` option is provided
- **R86:** Variables created by View and ViewList (which already set `elementId` in properties) must continue to work — their explicit value takes precedence

## Feature: viewdef Variable Property
**Source:** specs/viewdef-variable-property.md

- **R87:** After a View renders, it must set a `viewdef` property on its variable containing the resolved viewdef key
- **R88:** The `viewdef` property must be sent to the backend so the variable browser can display it
- **R89:** The `viewdef` property must update when the viewdef changes (type change triggers re-render)
- **R90:** ViewList item views must also record their viewdef (they render through the same View.render path)
- **R91:** When a view is cleared (type becomes empty), the `viewdef` property must be cleared

## Feature: Session Timers
**Source:** specs/session-defer.md

- **R92:** LuaSession must expose `session:setImmediate(fn)` returning an integer handle
- **R93:** LuaSession must expose `session:setTimeout(fn, ms)` returning an integer handle
- **R94:** LuaSession must expose `session:setInterval(fn, ms)` returning an integer handle
- **R95:** LuaSession must expose `session:clearImmediate(handle)`, `session:clearTimeout(handle)`, `session:clearInterval(handle)`
- **R96:** All timer functions must return immediately without blocking
- **R97:** Scheduled functions must run with the `session` global set (same context as ExecuteInSession)
- **R98:** Change detection (afterBatch) must run after each scheduled function invocation
- **R99:** `setImmediate` must run in the next ChanSvc turn; multiple calls queue in order
- **R100:** `setTimeout` must run once after the specified delay in milliseconds
- **R101:** `setInterval` must run repeatedly at the specified interval in milliseconds
- **R102:** Clear functions must cancel a pending timer by handle; no-op if already fired or cancelled
- **R103:** LuaSession must capture `tracker.ComputingVar` at schedule-time and restore it before running
- **R104:** Before running a scheduled function, `savedVar.Diags` must be set to nil
- **R105:** If a scheduled function errors, `savedVar.Error` must be set using `changetracker.DeferredCode` error type
- **R106:** After a scheduled function runs, `tracker.ComputingVar` must be set to nil
- **R107:** Server must provide `ExecuteInSessionAsync` — a fire-and-forget variant using `Svc` instead of `SvcSync`
- **R108:** If the session is destroyed before a timer fires, the timer must be silently cancelled
- **R109:** On session Shutdown, all active timers must be cancelled
- **R110:** (inferred) LuaSession must accept an `onDefer` callback set by the Server to decouple session from server
- **R111:** (inferred) LuaSession must maintain a timer registry with sequential integer handles
- **R115:** On Shutdown, the executor goroutine must finish any in-flight work item and stop before the Lua VM is closed (`State.Close()`); the wait is bounded by a timeout, after which Shutdown logs and proceeds
- **R116:** After shutdown is signaled (`done` closed), the executor must not begin executing a work item already queued on the executor channel; it returns a shutdown error to the caller instead
- **R117:** A deferred-task wrapper must check the shutdown signal before touching the Lua VM and skip execution if shutdown has begun
- **R118:** `Server.Shutdown` must not hold the sessions lock while invoking a session's Shutdown (snapshot under the lock, release, then shut down) so an in-flight deferred task that re-enters the Server cannot deadlock the executor join

## Feature: Destroy Response Batching
**Source:** specs/destroy-response-batching.md

- **R112:** Backend destroy notifications must be queued through the session's OutgoingBatcher, not sent directly to the WebSocket
- **R113:** (inferred) The Handler must have access to a message queuing interface so it can route destroy notifications through the batcher
- **R114:** (inferred) The OutgoingBatcher's throttle behavior (start timer on first queue, accumulate on subsequent queues) must batch all destroy responses from a single incoming batch into one outgoing frame

## Feature: App Discovery
**Source:** specs/app-discovery.md

- **R119:** When a site directory is configured and does not exist, the server refuses to start and returns an error naming the path; it does not panic
- **R120:** The site's `viewdefs/` directory is optional; its absence is not logged
- **R121:** An app is active when `lua/<app>` exists and resolves to a directory; plain files directly in `lua/` are not apps
- **R122:** Everything the server reads for an app is read through its `lua/<app>` entry, and the server never tests whether that entry is a link (so a Windows junction, which Go reports as a directory, counts as an app)
- **R123:** Viewdef sources rank in three tiers: built-in apps, then user apps (every other active app), then the site's own `viewdefs/`; apps rank alphabetically by name within a tier
- **R124:** The built-in app names come from a list the host supplies in the server configuration, programmatically only (not read from the config file); with no list there are no built-in apps, and a listed name that is not active is ignored
- **R125:** The server registers viewdefs from each active app's `viewdefs/` directory and from the site's `viewdefs/`, in rank order; a viewdef's name is its file name without `.html`; an app with no `viewdefs/` contributes none, silently
- **R126:** When two sources ship a viewdef of the same name, the earlier-ranked one wins and the later one is not registered, independent of load, link, or file-system walk order
- **R127:** Each viewdef-name collision is logged, naming the viewdef, the winning source, and the losing source
- **R128:** On-demand viewdef loading (a `TYPE.*.html` for a type first seen after startup) searches the same sources in the same order under the same collision rule
- **R129:** With hot-loading enabled, the server watches `lua/`; when an app is activated or deactivated there it re-runs discovery, updating the per-app watches, re-registering viewdefs, and re-resolving collisions
- **R130:** Deactivating an app stops new loads and hot reloads from it but does not unload Lua code already loaded from it
- **R137:** Two viewdef paths that resolve to the same file are not a collision and are not logged; the higher-ranked path registers as usual
- **R131:** A site `viewdefs/` directory created or removed while the server runs need not be noticed until the next start
- **R141:** The live viewdef collisions can be listed; each entry names the viewdef, the winning source and its file, and the losing source and its file, ordered by viewdef name then by the losing source's rank
- **R142:** The collision list is the current set: re-discovery recomputes it, so a fixed collision drops out, and a collision first met at run time (a hot-reloaded file whose name a higher-ranked source owns) joins it; the log still reports each collision once per server run
- **R143:** `(*cli.Server).ViewdefCollisions() []cli.ViewdefCollision` returns a running server's collision list; `cli.ViewdefCollision` has the fields `Key`, `Winner`, `WinnerPath`, `Loser`, `LoserPath`, a source named by its app name or `(site)`
- **R144:** `cli.ResolveViewdefCollisions(siteDir, builtins)` computes the collision list from disk without a server by discovering the site's apps and registering their viewdefs through the same code a starting server runs, so the two agree for the same files
- **R145:** `ui.viewdefCollisions()` returns the server's live collision list to Lua as an array of tables with `key`, `winner`, `winnerPath`, `loser`, `loserPath`, in the list's order

- **R138:** Each active app's whole directory is served over HTTP at `/<app>/…`, read through its `lua/<app>` entry; nothing in the directory is withheld
- **R139:** A static request resolves in order: a session path; then `html/<path>` if that file exists (so a site file overrides an app file); then the active app named by the first path segment, serving the rest of the path from its directory; otherwise not found
- **R140:** Without a site directory there is no per-app serving

## Feature: Per-App Hot-Loading
**Source:** specs/main.md

- **R132:** With hot-loading enabled, the server watches each active app's directory through its `lua/<app>` entry: every `.lua` file in it and its `viewdefs/`
- **R133:** The site's `viewdefs/` directory is watched if it existed at startup
- **R134:** An app's directory is watched where it really lives, and a change to any `.lua` file under it is mapped back to the loaded file it belongs to, so an app whose files sit outside the site directory hot-reloads every file, not only its entry file

## Feature: Module Tracking and Unloading (per-app hot-loading revision)
**Source:** specs/module-tracking.md

- **R135:** unloadDirectory must clean up HotLoader state for the directory: watches and pendingReloads
- **R136:** unloadModule must clean up HotLoader state for the module file: watches and pendingReloads

## Feature: Portable OS Primitives
**Source:** specs/interfaces.md

- **R146:** `ui.mkdir(path)` creates the directory and any missing parents, treats an existing directory as success, and returns `true`, or `nil, err` on failure
- **R147:** `ui.isDir(path)` returns `true` when `path` resolves (following links) to a directory and `false` otherwise, including when it does not exist; it never fails
- **R148:** `ui.listDir(path)` returns every entry of the directory, hidden ones included, sorted by name, as an array of `{name, isDir}` tables with `isDir` following links, or `nil, err` on failure
- **R149:** `ui.base64Decode(text)` decodes standard base64 to a byte string, or returns `nil, err` on invalid input
- **R150:** `ui.exec(argv)` runs `argv[1]` with the remaining elements as its arguments, directly with no shell, waits for it, and returns its stdout; a failure to start or a non-zero exit returns `nil, err` with the exit status and the program's stderr in `err`
- **R151:** `ui.platform()` returns a table with `os` set to the server's `GOOS` and `arch` set to its `GOARCH`
