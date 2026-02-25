# Sequence: Session Timer Execution

**Source Spec:** session-defer.md
**Requirements:** R92, R93, R94, R95, R96, R97, R98, R99, R100, R101, R102, R103, R104, R105, R106, R107, R108, R109, R110, R111
**Use Case:** Deferred/timed Lua execution via setImmediate, setTimeout, setInterval

## Participants

- LuaCode: Lua script running on executor goroutine
- LuaSession: Per-session Lua environment (timer registry, ComputingVar capture)
- Server: ExecuteInSessionAsync, session lookup
- ChanSvc: Session operation serializer
- ExecutorGoroutine: Lua VM executor
- Tracker: Change tracker (ComputingVar, Diags, Error)

## Sequence: setImmediate

```
  LuaCode          LuaSession             Server              ChanSvc         Executor        Tracker
     |                  |                     |                   |                |               |
     |--setImmediate--->|                     |                   |                |               |
     |   (fn)           |                     |                   |                |               |
     |                  |--capture----------->|                   |                |               |
     |                  |  ComputingVar       |                   |                |               |
     |                  |                     |                   |                |               |
     |                  |--allocate handle--->|                   |                |               |
     |                  |  store in registry  |                   |                |               |
     |                  |                     |                   |                |               |
     |                  |--onDefer(wrapped)-->|                   |                |               |
     |                  |                     |--Svc(svc, fn)---->|               |               |
     |                  |                     |  [goroutine waits |               |               |
     |<--return handle--|                     |   for ChanSvc]    |               |               |
     |                  |                     |                   |                |               |
     |  [current execution completes, ChanSvc turn ends]         |                |               |
     |                  |                     |                   |                |               |
     |                  |                     |              [next turn]           |               |
     |                  |                     |                   |                |               |
     |                  |                     |                   |--check cancel->|               |
     |                  |                     |                   |  [not cancelled]               |
     |                  |                     |                   |                |               |
     |                  |                     |                   |           set Diags=nil------->|
     |                  |                     |                   |           set ComputingVar---->|
     |                  |                     |                   |                |               |
     |                  |                     |                   |--execute(fn)-->|               |
     |                  |                     |                   |                |--Lua fn()     |
     |                  |                     |                   |                |               |
     |                  |                     |                   |  [if error: set DeferredCode]->|
     |                  |                     |                   |           ComputingVar=nil---->|
     |                  |                     |                   |                |               |
     |                  |                     |                   |--afterBatch--->|               |
     |                  |                     |                   |                |               |
```

## Sequence: setTimeout

Same as setImmediate, except step 3 uses `time.AfterFunc(ms, func() { onDefer(wrapped) })`
instead of calling onDefer directly. The timer goroutine fires after the delay,
then the ChanSvc/executor path is identical.

## Sequence: setInterval

Same as setTimeout, except a goroutine with `time.Ticker` calls `onDefer(wrapped)` on each tick.
The ticker runs until `clearInterval(handle)` is called or the session shuts down.

## Sequence: clearTimeout/clearInterval/clearImmediate

```
  LuaCode          LuaSession
     |                  |
     |--clearXxx------->|
     |   (handle)       |
     |                  |--lookup registry[handle]
     |                  |  set cancelled = true
     |                  |  call stop() if present
     |                  |
     |<--return---------|
```

## Sequence: Session Shutdown

```
  Server           LuaSession
     |                  |
     |--Shutdown()----->|
     |                  |--for each timer in registry:
     |                  |    set cancelled = true
     |                  |    call stop() if present
     |                  |--close executor, clean up Lua state
```

## Notes

- The cancelled flag is checked inside the ChanSvc turn, before touching the Lua VM
- This handles the race where a timer fires just as clearXxx is called
- setImmediate queuing order is preserved because Svc spawns goroutines that send on ChanSvc in order
- ComputingVar may be nil at schedule-time (e.g., timer set during init); diagnostic context is skipped in that case
