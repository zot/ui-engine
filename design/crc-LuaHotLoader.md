# LuaHotLoader
**Requirements:** R129, R130, R132, R134, R135, R136

**Source Spec:** main.md "Hot-Loading System"

## Responsibilities

### Knows
- config: Config object (baseDir accessed via config.Server.Dir)
- luaDir: Path to the Lua scripts directory
- watcher: fsnotify watcher instance
- server: Reference to Server for session access
- appWatches: App name → directories watched under its entry path `lua/<app>`
- pendingReloads: Tracking key → (changed path, last event time), for debounce

### Does
- Start: Watch the lua directory, the apps directory, and every directory under each active app's entry path (dot-dirs skipped)
- Stop: Clean up watcher resources
- handleFileChange(path): Re-execute modified Lua file in sessions that have loaded it
- rediscover: On a create/remove of a `lua/<app>` entry, re-run AppDiscovery and add/drop app watches; Lua already loaded from a removed app stays loaded
- queueReload(path): debounce keyed by tracking key, merged by file identity (os.SameFile), so one file reached through two watched paths reloads once; identity rather than key because on Windows EvalSymlinks does not resolve junctions, so one file can carry two keys
- computeTrackingKey(absPath): Compute baseDir-relative path for file tracking (resolves symlinks)
- reloadFile(path, session): Check IsFileLoaded(trackingKey), set reloading flag, reload via LoadCode()
- triggerSessionRefresh(session): Execute empty function via ws.ExecuteInSession to run AfterBatch (pushes viewdef/variable changes)
- recoverPanic: Wrap Lua execution in panic recovery, log errors instead of crashing server
- CleanupModule(trackingKey): Remove watches and pendingReloads for a module file
- CleanupDirectory(dirPath): Remove watches and pendingReloads for all files in a directory

## Collaborators

- Server: Provides access to active LuaSessions via GetLuaSessions()
- LuaSession: Provides IsFileLoaded() check, RequireLuaFile() for reload, reloading flag
- WebSocketEndpoint: Provides ExecuteInSession() for triggering AfterBatch
- Config: Provides lua.hotload setting and verbosity for logging
- AppDiscovery: Active apps and their entry paths
- fsnotify: File system notification library

## Sequences

- seq-lua-hotload.md: File change detection and reload flow
- seq-server-startup.md: Hot-loader initialization during startup
- seq-app-discovery.md: Re-discovery when an app is activated or deactivated

## Notes

- Only active when `--hotload` is enabled (lua.hotload = true)
- Watches the lua directory (default: `lua/` or `<dir>/lua/`), the apps directory, and each active app through its entry
- **BaseDir-relative tracking**: Files tracked by resolved target path relative to baseDir
  - `apps/myapp/app.lua` (for files in apps/)
  - `lua/mcp.lua` (for files in lua/)
  - Symlinks resolved to target before computing tracking key
- **Only reloads files already loaded by session**: Checks `IsFileLoaded(trackingKey)` before reloading
- **reloading flag**: Sets `session.reloading = true` before reload, `false` after (Lua code can detect)
- **Links**: See cross-cutting concern "Hot-Loading Through Activation Entries" in design.md
- Sessions maintain state between reloads (Lua code should use hot-loading conventions)
- Uses fsnotify for cross-platform file watching
- Debounces rapid file changes to avoid multiple reloads
- **Session refresh**: After reload, triggers AfterBatch via ws.ExecuteInSession to push changes to browser
- **Panic recovery**: All Lua execution wrapped in recover() - panics logged as errors, server continues
