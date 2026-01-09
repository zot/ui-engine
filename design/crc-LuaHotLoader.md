# LuaHotLoader

**Source Spec:** deployment.md

## Responsibilities

### Knows
- luaDir: Path to the Lua scripts directory
- watcher: fsnotify watcher instance
- server: Reference to Server for session access
- symlinkTargets: Map of symlink paths to their resolved target directories
- watchedDirs: Set of directories currently being watched

### Does
- Start: Initialize file watcher on lua directory
- Stop: Clean up watcher resources
- handleFileChange(path): Re-execute modified Lua file in all active sessions
- resolveSymlinks: Scan lua directory for symlinks, resolve and watch target directories
- updateSymlinkWatches: When symlinks change, update watched directories accordingly
- reloadFile(path, session): Execute modified file in a specific LuaSession
- triggerSessionRefresh(session): Execute empty function via ws.ExecuteInSession to run AfterBatch (pushes viewdef/variable changes)
- recoverPanic: Wrap Lua execution in panic recovery, log errors instead of crashing server

## Collaborators

- Server: Provides access to active LuaSessions via GetLuaSessions()
- LuaSession: Receives re-executed Lua code via LoadFileAbsolute()
- WebSocketEndpoint: Provides ExecuteInSession() for triggering AfterBatch
- Config: Provides lua.hotload setting and verbosity for logging
- fsnotify: File system notification library

## Sequences

- seq-lua-hotload.md: File change detection and reload flow
- seq-server-startup.md: Hot-loader initialization during startup

## Notes

- Only active when `--hotload` is enabled (lua.hotload = true)
- Watches the lua directory (default: `lua/` or `<dir>/lua/`)
- On file change, re-executes the modified file in ALL active sessions
- **Symlink handling**: Also watches real (target) directories of symlinked files
- **Dynamic watch updates**: When symlinks are added/modified/removed, updates watched directories
- Sessions maintain state between reloads (Lua code should use hot-loading conventions)
- Uses fsnotify for cross-platform file watching
- Debounces rapid file changes to avoid multiple reloads
- **Session refresh**: After reload, triggers AfterBatch via ws.ExecuteInSession to push changes to browser
- **Panic recovery**: All Lua execution wrapped in recover() - panics logged as errors, server continues
