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
- handleFileChange(path): Re-execute modified Lua file in sessions that have loaded it
- resolveSymlinks: Scan lua directory for symlinks, resolve and watch target directories
- updateSymlinkWatches: When symlinks change, update watched directories accordingly
- reloadFile(path, session): Check IsFileLoaded(), set reloading flag, reload via RequireLuaFile()
- triggerSessionRefresh(session): Execute empty function via ws.ExecuteInSession to run AfterBatch (pushes viewdef/variable changes)
- recoverPanic: Wrap Lua execution in panic recovery, log errors instead of crashing server

## Collaborators

- Server: Provides access to active LuaSessions via GetLuaSessions()
- LuaSession: Provides IsFileLoaded() check, RequireLuaFile() for reload, reloading flag
- WebSocketEndpoint: Provides ExecuteInSession() for triggering AfterBatch
- Config: Provides lua.hotload setting and verbosity for logging
- fsnotify: File system notification library

## Sequences

- seq-lua-hotload.md: File change detection and reload flow
- seq-server-startup.md: Hot-loader initialization during startup

## Notes

- Only active when `--hotload` is enabled (lua.hotload = true)
- Watches the lua directory (default: `lua/` or `<dir>/lua/`)
- **Only reloads files already loaded by session**: Checks `IsFileLoaded()` before reloading (ignores new files)
- **reloading flag**: Sets `session.reloading = true` before reload, `false` after (Lua code can detect)
- **Symlink handling**: Also watches real (target) directories of symlinked files
- **Dynamic watch updates**: When symlinks are added/modified/removed, updates watched directories
- Sessions maintain state between reloads (Lua code should use hot-loading conventions)
- Uses fsnotify for cross-platform file watching
- Debounces rapid file changes to avoid multiple reloads
- **Session refresh**: After reload, triggers AfterBatch via ws.ExecuteInSession to push changes to browser
- **Panic recovery**: All Lua execution wrapped in recover() - panics logged as errors, server continues
