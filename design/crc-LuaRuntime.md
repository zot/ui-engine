# LuaRuntime

**Source Spec:** interfaces.md, deployment.md

## Responsibilities

### Knows
- luaDir: Directory for Lua files (site's lua/ directory)
- luaSessions: Map of session ID to LuaSession instances
- executorChan: Channel for zero-arg functions (thread-safe execution)

### Does
- initialize: Start executor goroutine for thread-safe Lua execution
- startExecutor: Create goroutine that reads from executorChan and executes functions
- execute: Queue zero-arg function on executorChan (blocks until complete)
- createLuaSession: Create new LuaSession for frontend session, expose as global `session`, load main.lua
- destroyLuaSession: Clean up LuaSession when frontend session ends
- getLuaSession: Get LuaSession by session ID
- loadFile: Load and execute Lua file (via executor)
- loadCode: Load and execute Lua code string (via executor)
- afterBatch: Trigger change detection for a session after processing message batch
- shutdown: Close executor channel, clean up all Lua sessions

## Collaborators

- LuaSession: Per-frontend-session Lua API (created on new session)
- SessionManager: Notifies LuaRuntime when frontend sessions start/end
- VariableStore: LuaSessions use to create/manage variables
- PathNavigator: Used for path-based action dispatch

## Sequences

- seq-lua-session-init.md: Creating Lua session when frontend connects
- seq-lua-execute.md: Thread-safe Lua execution via executor channel
- seq-load-lua-code.md: Loading Lua code (file or dynamic via lua property)
- seq-lua-handle-action.md: Handling user actions via path-based method dispatch
- seq-backend-refresh.md: Automatic change detection after message batch

## Notes

**Backend Modes (see interfaces.md):**
- **Embedded Lua only** (`--lua`, no backend): LuaRuntime handles all backend logic
- **Connected backend only** (`--no-lua`): LuaRuntime disabled, BackendSocket handles logic
- **Hybrid** (`--lua` + backend): LuaRuntime provides reusable UI behaviors, backend provides app-specific logic

**Automatic Change Detection:**
- After processing a batch of messages, LuaRuntime triggers change detection
- Each LuaSession tracks object references for its watched variables
- The session computes current JSON values from live Lua objects
- Changed values are automatically sent to frontend watchers
- No manual update() calls needed in Lua code
