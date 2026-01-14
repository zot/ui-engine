# LuaSession

**Source Spec:** libraries.md, interfaces.md, protocol.md, deployment.md

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
- loadedModules: Lua table tracking loaded files (shared by require() and RequireLuaFile)
- reloading: Boolean flag (on sessionTable) - true during hot-reload, false otherwise

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
- prototype(name, init): Declare/update prototype with instance field tracking (see below)
- create(prototype, instance): Create tracked instance with weak reference (see below)
- processMutationQueue: Process queued prototypes after file load (see seq-prototype-mutation.md)
- RequireLuaFile(filename): Load Lua file using unified load tracker (skips if already loaded)
- IsFileLoaded(filename): Check if a file has been loaded (used by hot-loader)
- registerRequire: Set up custom require() using loadedModules table with circularity handling

**Prototype Management (Hot-Loading Support):**

`session:prototype(name, init)`:
- Looks up `name` in `prototypeRegistry` (NOT Lua globals - enables dotted names like `"contacts.Contact"`)
- If not in registry: creates new prototype with `type = name`, `__index` set, default `:new()` method, stores in registry
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

**Deprecated Methods (replaced by automatic prototype mutation):**
- ~~newVersion~~: No longer needed - prototype queuing handles versioning
- ~~getVersion~~: No longer needed
- ~~needsMutation(obj)~~: No longer needed - mutation is automatic

## Collaborators

- Server: Creates and owns this LuaSession (one per frontend session)
- LuaBackend: Per-session backend for watch management and change detection
- luaTrackerAdapter: Implements VariableStore interface, routes to per-session tracker
- WrapperRegistry: Provides wrapper factories for ui.registerWrapper
- LuaHotLoader: Re-executes modified Lua files via RequireLuaFile(), checks IsFileLoaded()

## Sequences

- seq-lua-session-init.md: Session creation and main.lua execution
- seq-load-lua-code.md: Dynamic code loading via lua property on variable 1
- seq-lua-handle-action.md: Action handling via path-based method dispatch
- seq-lua-hotload.md: Hot-loading re-executes modified Lua files with prototype management
- seq-prototype-mutation.md: Post-load mutation processing for schema migrations

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
  - `session:prototype(name, init)` - declare/update prototype, returns it for local assignment
  - Prototypes stored in session registry (not globals), enabling dotted names like `"contacts.Contact"`
  - `session:create(prototype, instance)` - create tracked instance with weak reference
  - `Prototype:mutate()` - optional migration method called automatically on instances
  - Post-load processing iterates mutation queue, calls mutate(), nils removed fields
- **Unified Load Tracking**: `loadedModules` Lua table shared by `require()` and `RequireLuaFile()`:
  - Circularity handling: mark loaded before execute, unmark on error
  - `session.reloading` flag: true during reload, false after (Lua code can detect hot-reload)
  - `IsFileLoaded(filename)` lets hot-loader skip files not yet loaded by session
