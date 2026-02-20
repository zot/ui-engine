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
- **R13:** unloadDirectory must clean up HotLoader state: watchers, symlinkTargets, pendingReloads for the directory
- **R14:** LuaSession must have an unloadModule(moduleName) method exposed to Lua as session:unloadModule(name)
- **R15:** unloadModule must remove the module's entry from loadedModules
- **R16:** unloadModule must remove prototypes registered by the module from prototypeRegistry (using RemovePrototype)
- **R17:** unloadModule must remove presenter types registered by the module from presenterTypes
- **R18:** unloadModule must remove wrappers registered by the module from wrapperRegistry
- **R19:** unloadModule must remove viewdefs registered by the module from viewdefManager
- **R20:** unloadModule must clean up HotLoader state for the module file: watches, symlinkTargets, pendingReloads
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

## Feature: Null View Clearing
**Source:** specs/viewdefs.md (Null view clearing)

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
