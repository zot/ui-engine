# Sequence: Require Lua File

**Source Spec:** libraries.md, module-tracking.md
**Requirements:** R21

## Participants

- LuaCode: Lua script calling require() or session code calling RequireLuaFile
- LuaSession: Per-session Lua environment
- Module: Tracks resources registered during load
- FileSystem: File system access
- Bundle: Embedded file bundle (for bundled binaries)

## Flow: require("modulename")

```
LuaCode                    LuaSession                      Module              FileSystem/Bundle
   |                           |                              |                     |
   |--require("foo.bar")------>|                              |                     |
   |                           |                              |                     |
   |                           |--check loadedModules["foo.bar"]                    |
   |                           |   [if cached, return cached value]                 |
   |                           |                              |                     |
   |                           |--convert to path: "foo/bar.lua"                    |
   |                           |                              |                     |
   |                           |--try filesystem--------------|----------------->   |
   |                           |   luaDir + "foo/bar.lua"     |      ReadFile       |
   |                           |                              |                     |
   |                           |   [if file exists]           |                     |
   |                           |<--content--------------------|---------------------|
   |                           |                              |                     |
   |                           |--ComputeTrackingKey()        |                     |
   |                           |   → "lua/foo/bar.lua"        |                     |
   |                           |                              |                     |
   |                           |   [if fs fails, try bundle]  |                     |
   |                           |--bundle.ReadFile-------------|----------------->   |
   |                           |   "lua/foo/bar.lua"          |                     |
   |                           |                              |                     |
   |                           |--mark loaded BEFORE execute  |                     |
   |                           |   loadedModules[key] = true  |                     |
   |                           |                              |                     |
   |                           |--SetCurrentModule----------->|                     |
   |                           |   (trackingKey, directory)   |  [create Module]    |
   |                           |                              |                     |
   |                           |--DoString(code)              |                     |
   |                           |                              |                     |
   |                           |   [during execution,         |                     |
   |                           |    prototype/presenter/      |                     |
   |                           |    wrapper registrations     |                     |
   |                           |    tracked to currentModule] |                     |
   |                           |                              |                     |
   |                           |--ClearCurrentModule--------->|                     |
   |                           |                              |                     |
   |                           |--cache result                |                     |
   |                           |   loadedModules[key] = result|                     |
   |                           |                              |                     |
   |<--return module value-----|                              |                     |
   |                           |                              |                     |
```

## Flow: DirectRequireLuaFile(filename)

```
Caller                     LuaSession                      Module              FileSystem
   |                           |                              |                     |
   |--DirectRequireLuaFile---->|                              |                     |
   |   ("apps/myapp/app.lua")  |                              |                     |
   |                           |                              |                     |
   |                           |--resolve absolute path       |                     |
   |                           |   [try luaDir, then baseDir] |                     |
   |                           |                              |                     |
   |                           |--ComputeTrackingKey()        |                     |
   |                           |   → "apps/myapp/app.lua"     |                     |
   |                           |                              |                     |
   |                           |--check loadedModules[key]    |                     |
   |                           |   [if loaded, return nil]    |                     |
   |                           |                              |                     |
   |                           |--ReadFile--------------------|----------------->   |
   |                           |                              |                     |
   |                           |<--content--------------------|---------------------|
   |                           |                              |                     |
   |                           |--mark loaded BEFORE execute  |                     |
   |                           |   loadedModules[key] = true  |                     |
   |                           |                              |                     |
   |                           |--SetCurrentModule----------->|                     |
   |                           |   (key, "apps/myapp")        |  [create Module]    |
   |                           |                              |                     |
   |                           |--DoString(code)              |                     |
   |                           |                              |                     |
   |                           |   [resources tracked         |                     |
   |                           |    to currentModule]         |                     |
   |                           |                              |                     |
   |                           |--ClearCurrentModule--------->|                     |
   |                           |                              |                     |
   |<--return (nil, nil)-------|                              |                     |
   |                           |                              |                     |
```

## Error Handling

```
LuaCode                    LuaSession                      Module
   |                           |                              |
   |--require("broken")------->|                              |
   |                           |                              |
   |                           |--mark loaded                 |
   |                           |--SetCurrentModule----------->|
   |                           |--DoString(code)              |
   |                           |                              |
   |                           |   [execution fails]          |
   |                           |                              |
   |                           |--ClearCurrentModule          |
   |                           |--unmark: loadedModules[key] = nil
   |                           |--delete modules[key]-------->|  [cleanup]
   |                           |--remove from moduleDirectories
   |                           |                              |
   |<--raise error-------------|                              |
   |                           |                              |
```

## Notes

- **Circularity handling**: Files marked loaded BEFORE execution to handle circular requires
- **Error recovery**: On execution failure, loadedModules entry cleared (allows retry), module tracking cleaned up
- **Module tracking**: `currentModule` set during load enables resource tracking
- **Tracking key**: BaseDir-relative path, symlinks resolved (e.g., `lua/foo.lua`, `apps/myapp/app.lua`)
- **Bundle fallback**: If filesystem fails, tries embedded bundle (for bundled binaries)
- **RequireLuaFile**: Wrapper that runs DirectRequireLuaFile via executor (thread-safe)
