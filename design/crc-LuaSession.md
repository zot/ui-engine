# LuaSession

**Source Spec:** libraries.md, interfaces.md, protocol.md, deployment.md, remove-prototype.md, module-tracking.md
**Requirements:** R1, R2, R3, R4, R5, R6, R7, R9, R10, R11, R12, R13, R14, R15, R16, R17, R18, R19, R20, R21

## Responsibilities

### Knows
- ID: Vended session ID (compact integer string like "1", "2", etc.) for backend communication
- State: Lua VM state for this session (isolated per-session)
- sessionTable: The session object exposed to Lua code as global `session`
- appVariableID: Variable 1 ID for this session (set by Lua code)
- appObject: Reference to the app Lua object (live table, not a wrapper)
- variableStore: Interface for session variable operations
- wrapperRegistry: Registry for wrapper factories
- executorChan: Channel for thread-safe Lua execution
- presenterTypes: Map of registered presenter types
- prototypeRegistry: Map of prototype name to stored init copy (for change detection)
- instanceRegistry: Map of prototype to weak set of instances (for mutation)
- mutationQueue: FIFO queue of (prototype, removedKeys) pairs pending mutation
- config: Config object (baseDir accessed via config.Server.Dir)
- loadedModules: Lua table tracking loaded files by baseDir-relative path (shared by require() and RequireLuaFile)
- reloading: Boolean flag (on sessionTable) - true during hot-reload, false otherwise
- luaDir: Path to lua/ directory (for loading files)
- modules: Map of tracking key to Module instance (tracks per-module resources)
- moduleDirectories: Map of directory path to list of Module instances
- currentModule: The Module currently being loaded (set during require/RequireLuaFile)
- hotLoaderCleanup: Callback function to clean up HotLoader state for a module/directory

### Does
- CreateLuaSession(vendedID): Initialize session, create session table, load main.lua
- createAppVariable: Create variable 1, store reference to Lua object for change detection
- getApp: Return the actual Lua app object (the live table, not a wrapper)
- createVariable: Create child variable with parent object reference
- destroyVariable: Destroy variable by ID (supports object reference lookup)
- GetLuaSession(vendedID): Return self if vendedID matches (per-session isolation)
- NotifyPropertyChange: Notify Lua watchers of property changes
- HandleFrontendCreate: Handle path-based variable creation from frontend
- HandleFrontendUpdate: Handle updates to path-based variables from frontend
- ExecuteInSession: Execute function within session context (sets global 'session')
- AfterBatch: Trigger change detection and return updates after message batch
- Shutdown: Close executor channel, clean up Lua state
- prototype(name, init, base): Declare/update prototype with instance field tracking (see below)
- create(prototype, instance): Create tracked instance with weak reference (see below)
- processMutationQueue: Process queued prototypes after file load (see seq-prototype-mutation.md)
- RemovePrototype(name, children): Remove prototype from registry; if children=true, also removes NAME.* children
- RequireLuaFile(filename): Load Lua file using unified load tracker (skips if already loaded); sets currentModule for resource tracking
- DirectRequireLuaFile(filename): Load file relative to baseDir, track by resolved baseDir-relative path; sets currentModule for resource tracking
- IsFileLoaded(trackingKey): Check if a file has been loaded by baseDir-relative key (used by hot-loader)
- registerRequire: Set up custom require() using loadedModules table with circularity handling
- resolveTrackingKey(path): Resolve symlinks and compute baseDir-relative path for file tracking
- unloadDirectory(name): Unload all modules in a directory and clean up HotLoader state
- unloadModule(moduleName): Remove all tracking related to a module (Lua exposed as session:unloadModule)
- setCurrentModule(trackingKey): Set the module being loaded for resource tracking
- clearCurrentModule(): Clear the current module after load completes

**Prototype Management (Hot-Loading Support):**

`session:prototype(name, init, base)`:
- `base` is optional; if nil, defaults to registered "Object" prototype (if exists), otherwise no metatable
- Looks up `name` in `prototypeRegistry` (NOT Lua globals - enables dotted names like `"contacts.Contact"`)
- If not in registry: creates new prototype with `type = name`, metatable set to resolved base (if any), stores in registry
- Adds default `:new()` only if no metatable (base provides `:new()` via inheritance)
- Stores shallow copy of `init` for change detection (preserves EMPTY markers)
- Creates instance tracking for this prototype (weak set)
- If already in registry and `init` differs from stored copy: updates prototype in place, computes removed fields, queues for mutation
- **Returns the prototype** (caller assigns to global: `Person = session:prototype("Person", {...})`)
- EMPTY global (`{}`) marks fields that start nil but are tracked for mutation

`session:create(prototype, instance)`:
- If `instance` is nil, creates empty table
- Sets metatable to prototype
- Registers instance in prototype's weak set for tracking
- Returns the instance

`RemovePrototype(name string, children bool)`:
- Removes prototype with `name` from `prototypeRegistry`
- Removes corresponding entry from `instanceRegistry`
- If `children` is true, also removes prototypes whose name starts with `NAME.` (e.g., "contacts.Person", "contacts.Address")
- Returns silently if prototype doesn't exist (no error)
- Existing instances retain their metatables (no instance destruction)

**Deprecated Methods (replaced by automatic prototype mutation):**
- ~~newVersion~~: No longer needed - prototype queuing handles versioning
- ~~getVersion~~: No longer needed
- ~~needsMutation(obj)~~: No longer needed - mutation is automatic

## Collaborators

- Server: Creates and owns this LuaSession (one per frontend session)
- LuaBackend: Per-session backend for watch management and change detection
- luaTrackerAdapter: Implements VariableStore interface, routes to per-session tracker
- WrapperRegistry: Provides wrapper factories for ui.registerWrapper
- LuaHotLoader: Re-executes modified Lua files via RequireLuaFile(), checks IsFileLoaded(), provides cleanup callback
- Module: Tracks resources registered by each module for cleanup during unload

## Sequences

- seq-lua-session-init.md: Session creation and main.lua execution
- seq-load-lua-code.md: Dynamic code loading via lua property on variable 1
- seq-lua-handle-action.md: Action handling via path-based method dispatch
- seq-lua-hotload.md: Hot-loading re-executes modified Lua files with prototype management
- seq-prototype-mutation.md: Post-load mutation processing for schema migrations
- seq-require-lua-file.md: require() and RequireLuaFile flow with module tracking
- seq-unload-module.md: Module and directory unloading

## Notes

- **Per-Session Isolation**: Each frontend session gets its own LuaSession with its own Lua VM state
- **Type Alias**: `type Runtime = LuaSession` for backward compatibility
- main.lua is responsible for creating variable 1 via session:createAppVariable()
- The `lua` property on variable 1 is automatically watched for dynamic code loading
- Action dispatch uses path resolution to call methods on presenter objects (NOT registered handlers)
- **Vended IDs**: LuaSession.ID is the vended ID (e.g., "1") not the internal UUID; saves bandwidth in backend communication
- **Server Owns Sessions**: Server maintains `luaSessions map[string]*LuaSession` and creates/destroys sessions via callbacks
- **Implements PathVariableHandler**: Server routes HandleFrontendCreate/Update to per-session LuaSession
- **Hot-Loading Support**: Automatic prototype and instance management:
  - `session:prototype(name, init, base)` - declare/update prototype, returns it for local assignment
  - `base` optional: defaults to "Object" prototype if registered, otherwise no inheritance
  - Prototypes stored in session registry (not globals), enabling dotted names like `"contacts.Contact"`
  - `session:create(prototype, instance)` - create tracked instance with weak reference
  - `Prototype:mutate()` - optional migration method called automatically on instances
  - Post-load processing iterates mutation queue, calls mutate(), nils removed fields
- **Unified Load Tracking**: `loadedModules` Lua table shared by `require()` and `RequireLuaFile()`:
  - Files tracked by baseDir-relative resolved path (symlinks resolved to target)
  - Circularity handling: mark loaded before execute, unmark on error
  - `session.reloading` flag: true during reload, false after (Lua code can detect hot-reload)
  - `IsFileLoaded(trackingKey)` lets hot-loader skip files not yet loaded by session
  - Example keys: `lua/mcp.lua`, `apps/myapp/app.lua`, `apps/myapp/init.lua`
- **Module Tracking**: Per-module resource tracking for clean unloading:
  - `modules` map: tracking key → Module instance
  - `moduleDirectories` map: directory path → list of Module instances
  - `currentModule`: Set during file load to track resource registration
  - Resources (prototypes, presenterTypes, wrappers) automatically tracked to currentModule
- **Module Unloading**: `session:unloadModule(name)` and `session:unloadDirectory(name)`:
  - Removes module's prototypes via RemovePrototype
  - Removes module's presenter types from presenterTypes map
  - Removes module's wrappers from wrapperRegistry
  - Removes module's entry from loadedModules
  - Cleans up HotLoader state via callback (watchers, symlinkTargets, pendingReloads)
