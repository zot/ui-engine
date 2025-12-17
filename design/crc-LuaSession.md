# LuaSession

**Source Spec:** libraries.md, interfaces.md, protocol.md

## Responsibilities

### Knows
- sessionId: Vended session ID (compact integer string like "1", "2", etc.) for backend communication
- luaState: Lua VM state for this session
- appVariable: Reference to variable 1 (created by main.lua)

### Does
- createAppVariable: Create variable 1, store reference to Lua object for change detection
- getApp: Return the actual Lua app object (the live table, not a wrapper)
- createVariable: Create child variable with parent object reference
- destroyVariable: Destroy variable by ID

## Collaborators

- LuaBackend: Owns this LuaSession, manages watches and change detection
- LuaRuntime: Executor for thread-safe Lua operations

## Sequences

- seq-lua-session-init.md: Session creation and main.lua execution
- seq-load-lua-code.md: Dynamic code loading via lua property on variable 1
- seq-lua-handle-action.md: Action handling via path-based method dispatch

## Notes

- Each frontend session gets exactly one LuaSession (owned by LuaBackend)
- main.lua is responsible for creating variable 1 via session:createAppVariable()
- The `lua` property on variable 1 is automatically watched for dynamic code loading
- Action dispatch uses path resolution to call methods on presenter objects (NOT registered handlers)
- **Vended IDs**: LuaSession.ID is the vended ID (e.g., "1") not the internal UUID; saves bandwidth in backend communication
- **Relationship to LuaBackend**: LuaSession provides the Lua-side API; LuaBackend owns the change-tracker and watch management
