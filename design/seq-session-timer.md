# Sequence: Session Timer Execution

**Source Spec:** session-defer.md
**Requirements:** R92, R93, R94, R95, R96, R97, R98, R99, R100, R101, R102, R103, R104, R105, R106, R107, R108, R109, R110, R111, R117, R118, R260, R261, R262
**Use Case:** Deferred/timed Lua execution via setImmediate, setTimeout, setInterval

## Participants

- LuaCode: Lua script running on the session executor
- LuaSession: Per-session Lua environment (timer registry, ComputingVar capture)
- Server: ExecuteInSessionAsync, session lookup
- ChanSvc: the session executor, the only place Lua runs for the session
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
     |                  |  return if absent or already cancelled
     |                  |  set cancelled = true
     |                  |  call stop() if present
     |                  |
     |<--return---------|
```

## Sequence: Session Shutdown

The Lua VM is touched only on the session's executor. Shutdown must close the VM with
no work touching it, so it closes the VM only once that executor has drained. It never
waits itself, so it completes when the session is destroyed from inside its own executor.

```
  Server                       LuaSession                  ChanSvc (session executor)
     |                              |                          |
     |--snapshot session registry-->|                          |
     |  (SessionManager.Snapshot),  |                          |
     |  release lock, then:         |                          |
     |--2.1 Shutdown()------------->|                          |
     |                              |--close(done)  (marks shut down; wrappers skip the VM)
     |                              |--for each timer: cancel + stop()
     |                              |--timerRegistry = nil
     |                              |--2.2 closeWhenDrained(State.Close)
     |                              |      [a watcher waits on Drained()]
     |<--returns at once------------|                          |
     |--2.3 Stop() (session teardown, after the backend)------>|
     |                              |                          |--the closure in flight,
     |                              |                          |  if any, returns
     |                              |<--Drained() closes-------|
     |                              |--2.4 State.Close()  (VM freed; nothing can run)
```

Server holds the session registry's lock only to snapshot the session set, then releases it
before calling each `Shutdown()`, so a task that re-enters the Server (e.g. the
app-loader calling `ExecuteInSession`) is never blocked on that lock. If the executor has
not drained within a timeout, 2.4 logs once and keeps waiting: a wedged task leaks its VM
rather than having it closed under it (R261).

## Notes

- The cancelled flag is checked inside the ChanSvc turn, before touching the Lua VM
- This handles the race where a timer fires just as clearXxx is called
- setImmediate queuing order is preserved because a post to ChanSvc is a direct send, so posts from one goroutine run in the order made (closure-actor.md)
- ComputingVar may be nil at schedule-time (e.g., timer set during init); diagnostic context is skipped in that case
- Work posted after the session executor stops is dropped by it (R228, R262), so work queued during teardown never runs
- Each deferred-task wrapper re-checks `done` before touching the VM, so a deferred task scheduled just before shutdown does not run against a closing VM
