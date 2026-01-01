# LuaBackend

**Source Spec:** main.md (UI Server Architecture - Hosted Backend), protocol.md (Session-Based Communication)

## Responsibilities

### Knows
- session: Reference to owning Session
- tracker: Per-session change-tracker.Tracker instance (NOT global)
- resolver: Lua-aware resolver for path navigation and wrapper creation
- luaState: Lua VM state for this session
- watchCounts: Map of variable ID to observer count {varId -> count}
- watchers: Map of variable ID to watching connections {varId -> []connId}
- appVariable: Reference to variable 1 (created by main.lua)

### Does
- Watch: Add observer for variable, manage tally, register with tracker if new
- Unwatch: Remove observer, decrement tally, unregister from tracker if zero
- UnwatchAll: Remove all watches for a connection (cleanup on disconnect)
- DetectChanges: Call tracker.DetectChanges() to compute and send updates
- HandleMessage: Process create/destroy/update/watch/unwatch, dispatch to appropriate handler
- HandleCreate: Create variable with properties, set up wrapper if specified
- HandleDestroy: Remove variable and all children from tracker
- HandleUpdate: Update variable value/properties, trigger path resolution
- HandleWatch: Add watcher, send immediate update with current value
- HandleUnwatch: Remove watcher
- HandleAction: Dispatch action to Lua method via path resolution
- CreateAppVariable: Create variable 1, register with tracker
- LoadMainLua: Load and execute main.lua with session global
- Shutdown: Clean up Lua state and tracker

## Collaborators

- Session: Owns this LuaBackend, forwards protocol messages
- change-tracker.Tracker: Per-session change detection and update dispatch
- change-tracker.Resolver: Path navigation and wrapper creation for Lua objects
- MessageRelay: Sends updates to watching connections

## Sequences

- seq-session-create-backend.md: Creating LuaBackend when session starts
- seq-backend-watch.md: Watch subscription with per-session tracker
- seq-backend-detect-changes.md: Per-session change detection via tracker
- seq-lua-session-init.md: Loading main.lua and creating app variable

## Notes

- **Merges WatchManager**: This class absorbs WatchManager functionality (watchCounts, watchers maps)
- **Per-session tracker**: Each LuaBackend has its own change-tracker.Tracker instance
- **Variable ID scope**: Variable IDs in watchCounts/watchers are session-scoped (no conflicts between sessions)
- **Implements Backend interface**: Session holds Backend interface, LuaBackend is the hosted implementation
- **Replaces global WatchManager**: The bug was that WatchManager used global maps keyed by varID, but varIDs are only unique within a session
- **Thread safety**: All Lua operations go through executor channel (same pattern as LuaSession)
