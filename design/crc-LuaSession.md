# LuaSession

**Source Spec:** libraries.md, interfaces.md, protocol.md

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

## Collaborators

- Server: Creates and owns this LuaSession (one per frontend session)
- LuaBackend: Per-session backend for watch management and change detection
- luaTrackerAdapter: Implements VariableStore interface, routes to per-session tracker
- WrapperRegistry: Provides wrapper factories for ui.registerWrapper

## Sequences

- seq-lua-session-init.md: Session creation and main.lua execution
- seq-load-lua-code.md: Dynamic code loading via lua property on variable 1
- seq-lua-handle-action.md: Action handling via path-based method dispatch

## Notes

- **Per-Session Isolation**: Each frontend session gets its own LuaSession with its own Lua VM state
- **Type Alias**: `type Runtime = LuaSession` for backward compatibility
- main.lua is responsible for creating variable 1 via session:createAppVariable()
- The `lua` property on variable 1 is automatically watched for dynamic code loading
- Action dispatch uses path resolution to call methods on presenter objects (NOT registered handlers)
- **Vended IDs**: LuaSession.ID is the vended ID (e.g., "1") not the internal UUID; saves bandwidth in backend communication
- **Server Owns Sessions**: Server maintains `luaSessions map[string]*LuaSession` and creates/destroys sessions via callbacks
- **Implements PathVariableHandler**: Server routes HandleFrontendCreate/Update to per-session LuaSession
