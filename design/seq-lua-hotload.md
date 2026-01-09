# Sequence: Lua Hot-Loading

**Source Spec:** deployment.md, libraries.md (Prototype Management)
**Use Case:** Re-execute modified Lua files in active sessions when `--hotload` is enabled

## Participants

- FileSystem: Source of file change events
- LuaHotLoader: File watcher component
- Server: Main server (owns luaSessions map)
- LuaSession: Per-session Lua environment
- WebSocketEndpoint: Session executor (triggers AfterBatch)

## Sequence

```
     FileSystem        LuaHotLoader            Server         WebSocketEndpoint    LuaSession(s)
        |                   |                   |                   |                   |
        |---modify--------->|                   |                   |                   |
        |   lua/app.lua     |                   |                   |                   |
        |                   |                   |                   |                   |
        |                   |--[debounce 100ms]-|                   |                   |
        |                   |                   |                   |                   |
        |                   |--GetLuaSessions()->|                   |                   |
        |                   |                   |                   |                   |
        |                   |<--[]*LuaSession----|                   |                   |
        |                   |                   |                   |                   |
        |                   |--[for each session, with panic recovery]                  |
        |                   |                   |                   |                   |
        |                   |--IsFileLoaded(filename)-------------------------------------->|
        |                   |                   |                   |                   |
        |                   |<--bool (skip if false)----------------------------------------|
        |                   |                   |                   |                   |
        |                   |--[set session.reloading = true]----------------------------->|
        |                   |                   |                   |                   |
        |                   |--RequireLuaFile(filename)------------------------------------>|
        |                   |                   |                   |                   |
        |                   |                   |                   |   [execute Lua    |
        |                   |                   |                   |    in session]    |
        |                   |                   |                   |                   |
        |                   |                   |                   |   [prototype()    |
        |                   |                   |                   |    queues changed |
        |                   |                   |                   |    prototypes]    |
        |                   |                   |                   |                   |
        |                   |<--ok/error-------------------------------------------------|
        |                   |                   |                   |                   |
        |                   |--[set session.reloading = false]---------------------------->|
        |                   |                   |                   |                   |
        |                   |--processMutationQueue()---------------------------------->|
        |                   |                   |                   |                   |
        |                   |                   |                   |   [see seq-       |
        |                   |                   |                   |    prototype-     |
        |                   |                   |                   |    mutation.md]   |
        |                   |                   |                   |                   |
        |                   |--ExecuteInSession(empty func)-------->|                   |
        |                   |                   |                   |                   |
        |                   |                   |                   |---[AfterBatch]--->|
        |                   |                   |                   |                   |
        |                   |                   |                   |   [detects and    |
        |                   |                   |                   |    pushes viewdef |
        |                   |                   |                   |    /var changes]  |
        |                   |                   |                   |                   |
        |                   |--[on panic: log error, continue]      |                   |
        |                   |                   |                   |                   |
        |                   |--[log reload result]                  |                   |
        |                   |                   |                   |                   |
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

## Prototype Declaration Pattern

**New pattern (automatic tracking):**
```lua
-- Declare prototype with instance fields
Person = session:prototype("Person", {
    name = "",
    email = "",
})

-- Shared state assigned separately (guarded)
Person.nextId = Person.nextId or 0

-- Optional: migration method for schema changes
function Person:mutate()
    self.email = self.email or ""
end
```

**Old pattern (deprecated):**
```lua
-- Manual preservation (no longer needed)
MyApp = MyApp or {type = "MyApp"}
```

## Notes

- **Debouncing**: File changes are debounced (100ms) to handle editors that write files in multiple steps
- **Only loaded files**: Checks `IsFileLoaded()` - skips files not yet loaded by session (new files ignored)
- **reloading flag**: `session.reloading` set `true` before reload, `false` after - Lua code can detect hot-reload
- **Symlink Transparency**: Changes to symlink targets reload as if the symlink file changed
- **Panic Recovery**: All Lua execution wrapped in recover() - panics logged as errors, server continues
- **Session Refresh**: After reload, ExecuteInSession(empty func) triggers AfterBatch which pushes viewdef/variable changes to browser
- **Prototype Management**: `session:prototype()` automatically:
  - Preserves prototype table identity (instances keep working)
  - Detects schema changes by comparing init to stored copy
  - Queues changed prototypes for mutation processing
- **Post-Load Processing**: After file execution, `processMutationQueue()` runs (see seq-prototype-mutation.md)
- **Guard App Creation**: Still needed to avoid recreating variable 1:
  ```lua
  if not session:getApp() then
    session:createAppVariable(App:new())
  end
  ```
