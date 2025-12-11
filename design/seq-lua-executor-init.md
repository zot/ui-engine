# Sequence: Lua Executor Initialization

**Source Spec:** interfaces.md
**Use Case:** Initializing LuaRuntime executor goroutine when --lua is enabled

## Participants

- Main: Server entry point
- LuaRuntime: Lua runtime manager

## Sequence

```
     Main                LuaRuntime
        |                      |
        |---initLua(path)----->|
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
        |<--luaRuntime---------|
        |                      |
```

## Notes

- Executor goroutine ensures single-threaded Lua access
- **No variable 1 created at startup** - each LuaSession creates its own variable 1
- LuaRuntime manages per-session LuaSessions (created when frontend connects)
- All Lua operations go through executorChan for thread safety
- See seq-lua-session-init.md for per-session Lua initialization
