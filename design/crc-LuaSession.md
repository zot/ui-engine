# LuaSession

**Source Spec:** libraries.md, interfaces.md, protocol.md

## Responsibilities

### Knows
- sessionId: Vended session ID (compact integer string like "1", "2", etc.) for backend communication
- luaState: Lua VM state for this session
- appVariable: Reference to variable 1 (created by main.lua)
- watchedVariables: Map of variable ID to referenced Lua object {varId -> luaTable}
- cachedValues: Map of variable ID to last JSON value sent {varId -> jsonString}
- propertyWatchers: Property watch callbacks {varId -> {property -> callback[]}}

### Does
- createAppVariable: Create variable 1, store reference to Lua object for change detection
- getApp: Return the actual Lua app object (the live table, not a wrapper)
- createVariable: Create child variable with parent object reference
- destroyVariable: Destroy variable by ID, remove from watched set
- watchProperty: Register callback for specific property changes on a variable
- notifyPropertyChange: Internal method to trigger property watchers on update
- computeAndSendChanges: Iterate watched variables, compute JSON, compare to cached, send updates for changes

## Collaborators

- LuaRuntime: Creates and manages LuaSession instances, triggers change detection after batch
- VariableStore: Backend for variable operations
- WatchManager: Watches lua property on variable 1 for dynamic code loading

## Sequences

- seq-lua-session-init.md: Session creation and main.lua execution
- seq-load-lua-code.md: Dynamic code loading via lua property on variable 1
- seq-lua-handle-action.md: Action handling via path-based method dispatch
- seq-backend-refresh.md: Automatic change detection after message batch

## Notes

- Each frontend session gets exactly one LuaSession
- main.lua is responsible for creating variable 1 via session:createAppVariable()
- The `lua` property on variable 1 is automatically watched for dynamic code loading
- Action dispatch uses path resolution to call methods on presenter objects (NOT registered handlers)
- **Vended IDs**: LuaSession.ID is the vended ID (e.g., "1") not the internal UUID; saves bandwidth in backend communication
- **Automatic change detection**: Variables store references to live Lua objects; after processing message batches, the framework computes current JSON values and sends updates for changes
- **No manual updates needed**: Backend code modifies Lua objects directly; changes are auto-detected
