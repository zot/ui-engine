# Sequence: Lua Hot-Loading

**Source Spec:** deployment.md
**Use Case:** Re-execute modified Lua files in active sessions when `--hotload` is enabled

## Participants

- FileSystem: Source of file change events
- LuaHotLoader: File watcher component
- Server: Main server (owns luaSessions map)
- LuaSession: Per-session Lua environment

## Sequence

```
     FileSystem        LuaHotLoader            Server            LuaSession(s)
        |                   |                   |                   |
        |---modify--------->|                   |                   |
        |   lua/app.lua     |                   |                   |
        |                   |                   |                   |
        |                   |--[debounce 100ms]-|                   |
        |                   |                   |                   |
        |                   |--GetLuaSessions()->|                   |
        |                   |                   |                   |
        |                   |<--[]*LuaSession----|                   |
        |                   |                   |                   |
        |                   |--[for each session]                   |
        |                   |                   |                   |
        |                   |--LoadFileAbsolute(path)-------------->|
        |                   |                   |                   |
        |                   |                   |   [execute Lua    |
        |                   |                   |    in session]    |
        |                   |                   |                   |
        |                   |<--ok/error-------------------------|
        |                   |                   |                   |
        |                   |--[log reload result]                  |
        |                   |                   |                   |
```

## Symlink Resolution Sequence

```
     FileSystem        LuaHotLoader            Server
        |                   |                   |
        |---create--------->|                   |
        |   lua/foo.lua     |                   |
        |   (symlink)       |                   |
        |                   |                   |
        |                   |--readlink()------>|
        |                   |   /real/path/foo.lua
        |                   |                   |
        |                   |--resolveDir()---->|
        |                   |   /real/path/     |
        |                   |                   |
        |                   |--[if not watched] |
        |                   |   watcher.Add(/real/path/)
        |                   |                   |
        |                   |--symlinkTargets[lua/foo.lua] = /real/path/
        |                   |                   |

(Later, target file changes)

     FileSystem        LuaHotLoader            Server            LuaSession(s)
        |                   |                   |                   |
        |---modify--------->|                   |                   |
        |  /real/path/foo.lua                   |                   |
        |                   |                   |                   |
        |                   |--[lookup symlink]-|                   |
        |                   |   lua/foo.lua     |                   |
        |                   |                   |                   |
        |                   |--[reload as lua/foo.lua]------------->|
        |                   |                   |                   |
```

## Notes

- **Debouncing**: File changes are debounced (100ms) to handle editors that write files in multiple steps
- **All Sessions**: Modified file is re-executed in ALL active sessions (not just one)
- **Symlink Transparency**: Changes to symlink targets reload as if the symlink file changed
- **Error Handling**: Errors in re-executed code are logged but don't crash the session
- **State Preservation**: Lua code should use hot-loading conventions:
  - `MyApp = MyApp or {type = "MyApp"}` (preserve existing prototypes)
  - `if not session:getApp() then ... end` (avoid recreating variable 1)
  - `session:needsMutation(obj)` (check if object needs schema migration)
