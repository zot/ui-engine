# Sequence: Lua Execute via Channel

**Source Spec:** interfaces.md
**Use Case:** Thread-safe Lua code execution via executor channel

## Participants

- Caller: Any goroutine (ProtocolHandler, MCPTool, etc.)
- LuaRuntime: Lua VM manager
- ExecutorGoroutine: Single goroutine for Lua execution
- LuaVM: Lua virtual machine

## Sequence

```
     Caller              LuaRuntime          ExecutorGoroutine            LuaVM
        |                      |                      |                      |
        |---execute(func)----->|                      |                      |
        |                      |                      |                      |
        |                      |---makeResultChan()--->|                      |
        |                      |                      |                      |
        |                      |---send(work)-------->|                      |
        |                      |   [executorChan]     |                      |
        |                      |                      |                      |
        |                      |                      |---receive(work)----->|
        |                      |                      |                      |
        |                      |                      |   [now in executor   |
        |                      |                      |    goroutine context]|
        |                      |                      |                      |
        |                      |                      |---func()------------>|
        |                      |                      |                      |
        |                      |                      |                      |---[lua code
        |                      |                      |                      |    executes]
        |                      |                      |                      |
        |                      |                      |<--result/error-------|
        |                      |                      |                      |
        |                      |                      |---send(result)------>|
        |                      |                      |   [resultChan]       |
        |                      |                      |                      |
        |                      |<--receive(result)----|                      |
        |                      |                      |                      |
        |<--result/error-------|                      |                      |
        |                      |                      |                      |
```

## Work Item Structure

```go
type luaWork struct {
    fn         func() (any, error)  // zero-arg function to execute
    resultChan chan luaResult       // channel for result
}

type luaResult struct {
    value any
    err   error
}
```

## Notes

- All Lua operations (callMethod, setProperty, loadCode) use execute()
- Caller blocks until execution completes (synchronous from caller's view)
- ExecutorGoroutine is the only goroutine that touches Lua VM state
- Prevents race conditions in multi-connection scenarios
- Channel buffer size 0 (unbuffered) for backpressure
- On shutdown, executorChan is closed, goroutine exits
