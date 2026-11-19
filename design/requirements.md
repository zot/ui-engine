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

## Feature: Session References and Rigs
**Source:** specs/session-ref.md

- **R153:** `Server.SessionRef(vendedID)` returns a reference and `true` for an existing session, and `false` when there is no such session
- **R154:** `SessionRef.ID()` returns the vended session ID
- **~~R155:~~** (Retired T6 — see R238) `SessionRef.Valid()` is true while the session's Lua session is running, and false after the session is destroyed or its Lua environment is restarted
- **R156:** A `SessionRef` never keeps its session alive and is safe to hold after the session ends
- **R157:** A `SessionRef` provides a Lua rig, a browser rig, an execution rig, and a `Station` holding all three
- **R158:** Every rig returns its reference (`Ref()`) and produces the session's `Station`
- **R159:** `ExecRig.Post(fn)` queues `fn` on the session's executor without waiting, runs it with the session's Lua context set, and runs no change detection afterward
- **R160:** `ExecRig.PostUpdate(fn)` queues `fn` like `Post`, and after it runs, runs change detection and pushes changes when a browser is connected
- **R161:** `Post` and `PostUpdate` return `false` and do nothing when the reference is invalid; posted work runs whether or not a browser is connected
- **R162:** `BrowserRig.Connected()` reports whether a browser connection is attached to the session
- **R163:** `BrowserRig.Update()` runs change detection and pushes changes immediately, and is called only on the session's executor
- **R164:** `LuaRig.Session()` returns the session's `LuaSession`, for use only on the session's executor
- **R165:** (inferred) Obtaining a rig or the station does no work beyond copying the reference
- **~~R166:~~** (Retired T3 — see R173) Session references are implemented over the existing session maps; `ExecuteInSession`, `ExecuteInSessionAsync` and the maps are unchanged
- **R167:** ui-engine exposes no notion of a current session; a host that has one supplies the reference
- **R168:** `Server.ExecuteInSession` keeps its signature and delegates all its behaviour to `Server.PrimitiveExecuteInSession(vendedID, fn, dangerous)` with `dangerous` true
- **R169:** `PrimitiveExecuteInSession` with `dangerous` false behaves exactly as `ExecuteInSession` did before dangerous execution was introduced
- **R170:** With `dangerous` true and `AllowDangerousExecution` set, `PrimitiveExecuteInSession` logs a warning naming the file and line that called `ExecuteInSession`, at most once per site for the life of the process, then proceeds as with `dangerous` false
- **R171:** With `dangerous` true and `AllowDangerousExecution` not set, `PrimitiveExecuteInSession` panics with a message naming the calling site, before queueing anything
- **R172:** The session config property `AllowDangerousExecution` (`allow_dangerous_execution` in the config file) defaults to set
- **~~R173:~~** (Retired T7 — see R229) Session references are implemented over the existing session maps; `ExecuteInSessionAsync` and the maps are unchanged
- **R229:** Each part of a session has exactly one home; the old way of reaching a part reads that home and keeps no copy
- **R230:** A session's Lua session is attached to the session while it is created, before `main.lua` runs, and does not change for the life of the session
- **R231:** No table other than the session holds a session's Lua session: not the server, and not the variable store's adapter
- **R232:** A Lua session is found by vended ID by going from the vended ID to the session and asking it; `Server.GetLuaSession` keeps its signature and works this way, as do browser messages, change detection and the debug page
- **R233:** Every running session is enumerated from the session registry, including for hot-loading and viewdef pushes
- **R234:** A destroyed session cannot be found by its vended ID from the moment its IDs are dropped, including while its Lua session is being torn down
- **R235:** `Server.ExecuteInSessionAsync` is `ExecRig.PostUpdate` on a reference to the session, and does nothing when there is no such session
- **R236:** `Server.ExecuteInSession` finds the session the same way a session reference does
- **R237:** No rig has a method that waits for the session's executor
- **R238:** `SessionRef.Valid()` is true while the session's Lua session is running, and false after the session is destroyed or a new session takes its vended ID
- **R239:** A session's backend is held by the session; the variable store's adapter reaches it through the session registry and keeps no copy
- **R240:** The counter for a session's server-created variable IDs (negative, from `-1`) belongs to that session's backend
- **R241:** Every variable operation on the variable store names its session (creating, reading a value or a property, updating, destroying); nothing maps a variable ID back to a session
- **R242:** A Lua session names its own session in every variable operation it makes
- **R243:** In embedded mode, `session:destroyVariable(id)` destroys that session's variable
- **R244:** A browser connection to an existing session holds that session, and the session records the connection; disconnecting undoes both together
- **R245:** The connection-to-session relationship is kept once in each direction: no other table maps a connection to its session or a session to its connections
- **R246:** Messages from a browser connection go to its session's executor; a connection whose session does not exist reaches nothing
- **R247:** Whether a browser is connected to a session, and which connections a broadcast to it reaches, come from the session; a destroyed session has no connections, so change detection does not run for it and a broadcast to it sends nothing
- **R248:** Nothing issues or checks a reconnect token; a browser can rejoin any session that has not timed out
- **R249:** Until registering presenters at URL paths (`interfaces.md`) is built, the server keeps no table of a session's URL paths

## Feature: Tailor
**Source:** specs/tailor.md

- **R174:** `ExecRig.Tailor(name, deadline)` creates a Tailor for one sewing operation on the rig's session
- **R175:** A Tailor created with deadline 0 is long-lived: it never expires
- **R176:** `Sew(fn, updateOnErr)`, `SewUpdate(fn)`, `End(fn)` and `EndUpdate(fn)` each post a stitch to the session executor and return without waiting
- **R177:** A stitch runs with the session's Lua context set; `Sew` and `End` stitches are passive, and `SewUpdate` and `EndUpdate` stitches run change detection afterward, pushing changes when a browser is connected
- **R178:** Posting a stitch records the current time and the calling site (file and line) in the Tailor's pending entry
- **R179:** Before posting, a stitch checks `IsValid()`; if the session has ended, the Tailor closes as failed and nothing is posted
- **R180:** A stitch runs `fn` with panics recovered, and a panic counts as an error
- **R181:** When a stitch's `fn` returns an error, the error is logged with the Tailor's name and the Tailor closes as failed
- **R182:** A `Sew` stitch whose `fn` errors runs change detection afterward only when `updateOnErr` is set
- **R183:** `End` and `EndUpdate` close the Tailor after `fn` runs, whether or not it errored
- **R184:** Posting a stitch after `End` or `EndUpdate` has been posted panics
- **R185:** Posting a stitch after the Tailor closed through a failure, its deadline, or its session ending logs once and posts nothing
- **R186:** `IsValid()` is true while the session is valid and the Tailor is not closed
- **R187:** `State()` returns the session's `*lua.LState` while a stitch of this Tailor is running, and panics otherwise
- **R188:** A Tailor never hands out the session's `LuaSession`
- **R189:** A Tailor with a non-zero deadline closes as failed when the deadline passes, logging its name and last calling site
- **R190:** `Context()` returns a context that is cancelled when the Tailor closes for any reason
- **R191:** A Tailor joins its session's pending list when created and leaves it when it closes
- **R192:** `ExecRig.PendingTailors()` lists the session's open Tailors with name, creation time, last stitch time, last calling site, and whether each is long-lived
- **R193:** When a session is destroyed, each of its open Tailors closes as failed, its context is cancelled, and it is logged with its name, age, and last calling site
- **R194:** `Tailor.Ref()` returns the Tailor's session reference

## Feature: Closure Actor
**Source:** specs/closure-actor.md

- **R195:** Closures posted to a `ChanSvc` by one goroutine run in the order that goroutine posted them, for fire-and-forget and synchronous posts alike
- **R196:** A post is a direct send on the calling goroutine; it is never handed to a freshly spawned goroutine
- **R197:** The `ChanSvc` loop goroutine never runs a posted closure; it keeps a queue and an in-flight flag, touched only on that goroutine, and each closure runs on a goroutine of its own that posts back when it finishes
- **R198:** Exactly one closure posted to a `ChanSvc` runs at any instant
- **R199:** A `ChanSvc` absorbs bursts in an unbounded queue; its channel is one slot deep
- **R200:** A closure panic is not recovered by the `ChanSvc`
- **R201:** A fire-and-forget post from inside a running closure queues behind it; a synchronous post from inside a running closure is forbidden
- **R202:** Popping the `ChanSvc` queue clears the popped slot before advancing
- **~~R203:~~** (Retired T4 — see R218) `ChanSvc.Stop` posts a sentinel behind everything already posted and never closes the channel
- **~~R204:~~** (Retired T5 — see R224) A session's executor is created and started on first use, and stopped when the session's executor is cleaned up
- **R218:** When a `ChanSvc` stops, a closure in flight runs to completion and closures still queued behind it are abandoned
- **R219:** After `ChanSvc.Stop` the loop goroutine has ended, and a fire-and-forget post is dropped and returns at once
- **R220:** After `ChanSvc.Stop` a synchronous post returns `ErrStopped` without waiting
- **R221:** A synchronous caller whose closure was abandoned in the queue by a stop returns `ErrStopped` instead of waiting forever
- **R222:** A closure finishing after its `ChanSvc` stopped reports back without blocking or panicking
- **R223:** Stopping a `ChanSvc` twice is harmless, and stopping it from inside a running closure is allowed
- **R224:** A session's executor is created and started when the session is created
- **R225:** A session's executor is stopped when the session is destroyed, after the session's backend has been torn down
- **R226:** Destroying a session from inside its own executor completes
- **R227:** After a session is destroyed, the goroutine count is what it was before the session was created
- **R228:** Work for a session that does not exist, including one already destroyed, creates no executor and never blocks its poster: a synchronous execution returns an error, and fire-and-forget work is dropped

## Feature: Host Operations
**Source:** specs/host-ops.md

- **R205:** `LuaRig.DefineHostOp(name, deadline, op)` installs `ui[name]` in the session's Lua state
- **R206:** Calling a host operation with a `target` that is not a table, or a `field` that is not a string, raises a Lua argument error at the call
- **R207:** A host operation call returns nothing, at once, without waiting for its work
- **R208:** A trailing function argument is taken as the callback and is not passed to `op`
- **R209:** `op` is called on the executor with the arguments after `target` and `field`, and returns the `work` to run
- **R210:** When `op` returns an error, the error is delivered as a failure and no work runs
- **R211:** Each call creates a Tailor named after the operation; `work` runs on its own goroutine with a context that is cancelled when the deadline passes or the session ends
- **R212:** The result is delivered in one terminal active stitch, which runs change detection afterward
- **R213:** On success, delivery sets `target[field]` to the result converted with `GoToLua` and sets `target[field .. "Error"]` to nil
- **R214:** On failure, delivery sets `target[field]` to nil and `target[field .. "Error"]` to the error message
- **R215:** After the fields are written, a callback is called as `cb(result, nil)` on success or `cb(nil, message)` on failure
- **R216:** A deadline of 0 means none; otherwise `work`'s context expires after the deadline, and the Tailor closes as failed after twice the deadline, delivering nothing if `work` has not returned

## Feature: Bundling
**Source:** specs/deployment.md

- **R217:** Bundling skips editor leftovers (names starting with `#` or `.#`, or ending with `~`) and fossil version-control state (`.fslckout`, `_FOSSIL_`, `*.fossil`)
