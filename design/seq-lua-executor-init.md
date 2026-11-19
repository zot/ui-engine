# Sequence: Lua Executor Initialization

**Source Spec:** interfaces.md, closure-actor.md
**Requirements:** R254, R260
**Use Case:** Creating a LuaSession and giving it the session's executor (per-session initialization)

## Participants

- Server: Main server (finds Lua sessions through the session registry)
- Session: owns the session's executor (a ChanSvc)
- LuaSession: Per-session Lua environment

## Sequence

```
     Server                 Session              LuaSession
        |                      |                      |
        |--1.1 NewRuntime(cfg, luaDir, vdm)---------->|
        |                      |                      |---NewState()
        |                      |                      |   [create Lua VM]
        |                      |                      |---loadStdlib()
        |                      |                      |---registerRequire()
        |                      |                      |---registerUIModule()
        |                      |                      |   [no goroutine; work
        |                      |                      |    from outside runs
        |                      |                      |    under a mutex until
        |                      |                      |    1.3]
        |<--luaSession--------------------------------|
        |                      |                      |
        |--1.2 attachLua(ls)-->|                      |
        |                      |                      |
        |--1.3 SetExecutor(run, closeWhenDrained)---->|
        |   run: SvcSync on the session's ChanSvc, no change detection
        |   closeWhenDrained: wait on the ChanSvc's Drained(), then close
        |                      |                      |
        |--1.4 CreateLuaSession(vendedID)------------>|
        |                      |   [main.lua compiled off the executor,
        |                      |    then run through run]
```

## Notes

- **Per-Session VM**: Each LuaSession has its own Lua state (complete isolation)
- **Note**: `type Runtime = LuaSession` exists in code for backward compatibility
- **One executor**: the session's ChanSvc is the only thing that runs Lua for the session. Work already on it runs Lua in place (`RunInSession`); work from outside posts through `run` (R254-R256)
- **Standalone sessions** (tests, no Server) keep the mutex from 1.1: one caller at a time, closed at once on Shutdown
- **No variable 1 created at startup** - main.lua creates it via session:createAppVariable()
- **Session owns its Lua session**: Server attaches each LuaSession to its Session and finds it through the session registry; it keeps no map of its own
- See seq-lua-session-init.md for full session initialization including main.lua loading
