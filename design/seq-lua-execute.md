# Sequence: Lua Work from Outside the Executor

**Source Spec:** interfaces.md, closure-actor.md
**Requirements:** R254, R255, R256
**Use Case:** Running Lua for a session from code that is not already on the session's executor

## Participants

- Caller: code off the executor (flib's WithLua, the hot-loader, session creation, output redirection)
- LuaSession: Per-session Lua environment
- ChanSvc: the session's executor (closure-actor.md)
- LuaVM: Lua virtual machine

## Sequence

```
     Caller              LuaSession              ChanSvc                LuaVM
        |                      |                      |                      |
        |--1.1 [prepare off the executor: read, compile, open files]         |
        |                      |                      |                      |
        |--1.2 ExecuteInSession(vendedID, fn)-------->|                      |
        |                      |--run(fn): SvcSync--->|                      |
        |                      |                      |--fn, with the        |
        |                      |                      |  session global set->|
        |                      |                      |                      |--[lua code
        |                      |                      |                      |   runs]
        |                      |<--result/error-------|                      |
        |<--result/error-------|                      |                      |
```

Work already on the executor (browser messages, change detection, timers, host-op
deliveries, rig work) never takes this path. It calls `RunInSession`, which runs `fn` in
place: a synchronous post from inside the executor would wait for itself (R255).

## Notes

- 1.1 is the sewing rule: only the Lua touch goes on the executor, because the executor serves every browser message, timer and change detection of its session (closure-actor.md)
- The caller blocks until execution completes; the executor does not
- A standalone LuaSession (no Server) runs `fn` under a mutex instead of a ChanSvc
- After the session's executor stops, the post is dropped and the caller gets an error (R228, R262)
