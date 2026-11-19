# Sequence: Lua Executor Initialization

**Source Spec:** interfaces.md
**Use Case:** Creating a LuaSession with executor goroutine (per-session initialization)

## Participants

- Server: Main server (finds Lua sessions through the session registry)
- LuaSession: Per-session Lua environment

## Sequence

```
     Server               LuaSession
        |                      |
        |--NewRuntime(cfg,     |
        |   luaDir, vdm)------>|
        |                      |
        |                      |---NewState()
        |                      |   [create Lua VM]
        |                      |
        |                      |---makeChan()
        |                      |   [executorChan]
        |                      |
        |                      |---startExecutor()
        |                      |   [goroutine reads
        |                      |    from channel]
        |                      |
        |                      |---loadStdlib()
        |                      |   [via executor]
        |                      |
        |                      |---registerRequire()
        |                      |   [custom module loader]
        |                      |
        |                      |---registerUIModule()
        |                      |   [ui.* API]
        |                      |
        |<--luaSession---------|
        |                      |
```

## Notes

- **Per-Session VM**: Each LuaSession has its own Lua state (complete isolation)
- **Note**: `type Runtime = LuaSession` exists in code for backward compatibility
- Executor goroutine ensures single-threaded Lua access per session
- **No variable 1 created at startup** - main.lua creates it via session:createAppVariable()
- **Session owns its Lua session**: Server attaches each LuaSession to its Session and finds it through the session registry; it keeps no map of its own
- All Lua operations go through executorChan for thread safety
- See seq-lua-session-init.md for full session initialization including main.lua loading
