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

## Feature: Server-to-Frontend Batching
**Source:** specs/protocol.md

- **R39:** Server must send batched messages to frontend as JSON arrays
- **R40:** Frontend must detect and handle incoming JSON array batches
- **R41:** Frontend must start outgoing batch timer BEFORE processing incoming messages
- **R42:** ServerOutgoingBatcher must group messages by connection and send one batch per connection
