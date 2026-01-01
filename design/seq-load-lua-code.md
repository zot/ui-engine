# Sequence: Load Lua Code

**Source Spec:** interfaces.md, libraries.md
**Use Case:** Loading Lua presenter logic into runtime

## Participants

- LuaSession: Per-session Lua environment
- WatchManager: Property watching
- FileSystem: File system access

## Sequence: Session main.lua Loading

See seq-lua-session-init.md for initial main.lua loading when a session starts.

## Sequence: Dynamic Code Loading via lua Property

```
     Frontend          ProtocolHandler           LuaSession          WatchManager          FileSystem
        |                      |                      |                      |                      |
        |---update(1,--------->|                      |                      |                      |
        |    {lua:"code"})     |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---updateProperty---->|                      |                      |
        |                      |   (sessionId, 1,     |                      |                      |
        |                      |    "lua", value)     |                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---notify(1,"lua")--->|                      |
        |                      |                      |                      |                      |
        |                      |                      |   [check value       |                      |
        |                      |                      |    format]           |                      |
        |                      |                      |                      |                      |
        |                      |                      |   [if ends with .lua]|                      |
        |                      |                      |---loadFile-----------|--------------------->|
        |                      |                      |   (lua/filename.lua) |                      |
        |                      |                      |                      |                      |
        |                      |                      |<--fileContents-------|----------------------|
        |                      |                      |                      |                      |
        |                      |                      |   [else: inline code]|                      |
        |                      |                      |---doString(code)     |                      |
        |                      |                      |                      |                      |
        |                      |                      |   [Lua code          |                      |
        |                      |                      |    executes]         |                      |
        |                      |                      |                      |                      |
        |                      |<--result/error-------|                      |                      |
        |                      |                      |                      |                      |
        |<--response-----------|                      |                      |                      |
        |                      |                      |                      |                      |
```

## Dynamic Loading Modes

The `lua` property on variable 1 supports two modes:

1. **Inline code**: Value is Lua code to evaluate directly
   ```json
   {"type": "update", "id": 1, "properties": {"lua": "print('hello')"}}
   ```

2. **File reference**: Value ends with `.lua`, loads from `<site>/lua/<filename>`
   ```json
   {"type": "update", "id": 1, "properties": {"lua": "helpers.lua"}}
   ```

## Notes

- **Session-scoped**: Code is loaded into the specific LuaSession for the session
- **Thread safety**: All code execution goes through LuaSession's executor channel
- **Watch mechanism**: Built-in watcher on variable 1's `lua` property triggers loading
- **File loading**: File references load from site's `lua/` directory
- Code executed in session's Lua VM
- Presenter objects defined in code have methods callable via ui-action path syntax
