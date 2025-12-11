# LuaSession

**Source Spec:** libraries.md, interfaces.md, protocol.md

## Responsibilities

### Knows
- sessionId: Vended session ID (compact integer string like "1", "2", etc.) for backend communication
- luaState: Lua VM state for this session
- appVariable: Reference to variable 1 (created by main.lua)
- variables: Cached LuaVariable wrappers {varId -> LuaVariable}
- propertyWatchers: Property watch callbacks {varId -> {property -> callback[]}}

### Does
- createAppVariable: Create variable 1 (the app variable) with initial value and properties
- getAppVariable: Return LuaVariable wrapper for variable 1 (error if not created)
- createVariable: Create child variable with parent ID, value, and properties
- getVariable: Return LuaVariable wrapper for variable by ID (cached)
- destroyVariable: Destroy variable by ID, remove from cache
- watchProperty: Register callback for specific property changes on a variable
- notifyPropertyChange: Internal method to trigger property watchers on update

## Collaborators

- LuaVariable: Variable wrapper objects returned by session methods
- LuaRuntime: Creates and manages LuaSession instances
- VariableStore: Backend for variable operations
- WatchManager: Watches lua property on variable 1 for dynamic code loading

## Sequences

- seq-lua-session-init.md: Session creation and main.lua execution
- seq-load-lua-code.md: Dynamic code loading via lua property on variable 1
- seq-lua-handle-action.md: Action handling via path-based method dispatch

## Notes

- Each frontend session gets exactly one LuaSession
- main.lua is responsible for creating variable 1 via session:createAppVariable()
- The `lua` property on variable 1 is automatically watched for dynamic code loading
- Action dispatch uses path resolution to call methods on presenter objects (NOT registered handlers)
- **Vended IDs**: LuaSession.ID is the vended ID (e.g., "1") not the internal UUID; saves bandwidth in backend communication
