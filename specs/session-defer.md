# Session Timers — setImmediate, setTimeout, setInterval

**Language:** Go, Lua
**Environment:** ui-engine backend (internal/lua, internal/server)

## Purpose

Allow Lua code to schedule functions for deferred or timed execution. This avoids the deadlock that occurs when Lua code (running on the session's executor) tries to call `ExecuteInSession` synchronously.

## Problem

Lua code runs on the session's executor, the one closure actor that serializes everything the session does: browser messages, external calls, and all access to its Lua state (`closure-actor.md`, "The session's Lua state"). A synchronous `ExecuteInSession` call from Lua would post to that executor and wait for it while it is busy running the caller — deadlock.

## API

### Lua

```lua
-- Immediate: runs in the next ChanSvc turn
local handle = session:setImmediate(function()
    -- full session context, afterBatch runs after
end)
session:clearImmediate(handle)

-- Delayed: runs after ms milliseconds
local handle = session:setTimeout(function()
    -- full session context, afterBatch runs after
end, ms)
session:clearTimeout(handle)

-- Repeating: runs every ms milliseconds
local handle = session:setInterval(function()
    -- full session context, afterBatch runs after each call
end, ms)
session:clearInterval(handle)
```

- All scheduling functions accept a zero-argument function and return an integer handle
- All return immediately without blocking
- The scheduled function runs with the `session` global set (same as `ExecuteInSession`)
- Change detection (`afterBatch`) runs after each invocation
- `setImmediate`: runs in the next ChanSvc turn; multiple calls queue in order
- `setTimeout`: runs once after the specified delay (milliseconds)
- `setInterval`: runs repeatedly at the specified interval (milliseconds)
- Clear functions cancel a pending timer by handle; no-op if already fired or cancelled

### Go

```go
// Server.ExecuteInSessionAsync — fire-and-forget variant of ExecuteInSession
func (s *Server) ExecuteInSessionAsync(vendedID string, fn func() (interface{}, error))
```

- Queues execution through ChanSvc using `Svc` (async) instead of `SvcSync` (blocking)
- Runs `fn` on the session's executor with the session context set
- Triggers `afterBatch` after execution
- Errors are logged, not returned (fire-and-forget)

## Mechanism

All three timer functions share the same execution path:

1. Capture the Lua function and the current `tracker.ComputingVar`
2. Allocate a handle from an incrementing counter; store in a registry with a `cancelled` flag
3. Schedule via the `onDefer` callback (set by Server), which calls `Server.ExecuteInSessionAsync`
4. Before executing, check the `cancelled` flag — skip if cancelled
5. The function runs on the session's executor with session context, then `afterBatch` pushes changes

Timing differences:
- **setImmediate**: calls `onDefer` directly (next ChanSvc turn)
- **setTimeout**: `time.AfterFunc(duration, func() { onDefer(...) })`
- **setInterval**: goroutine with `time.Ticker`; each tick calls `onDefer(...)`; stops when cancelled or session shuts down

## Diagnostic Context

When a timer function is called, the Go closure captures `tracker.ComputingVar` — the variable whose computation scheduled the timer. Before executing:

1. Set `savedVar.Diags = nil` (clear stale diagnostics)
2. Set `tracker.ComputingVar = savedVar` (attribute diagnostics to originating variable)
3. Run the Lua function
4. If the Lua function errors, set `savedVar.Error` using `changetracker.DeferredCode` error type
5. Set `tracker.ComputingVar = nil`

Errors become visible through the variable browser attributed to the originating variable.

## Handle Registry

LuaSession maintains a `timerRegistry map[int64]*timerEntry`:
- `timerEntry`: cancelled bool, stop func() (for setTimeout/setInterval cleanup)
- Handles are sequential integers (no reuse needed — int64 won't exhaust)
- Clear functions set `cancelled = true` and call `stop()` if present
- On session Shutdown, all active timers are cancelled

## Shutdown Safety

A session's Lua VM is touched only on the session's executor. `Shutdown` must free
that VM (`State.Close()`) with no work touching it, or the process crashes inside
the Lua VM. Deferred tasks (timers, and the app-loader a timer may invoke) are the
work most likely to be in flight when a session shuts down.

Shutdown never waits for the executor and never closes the VM itself, because it
can be called from work running on that executor (destroying a session from inside
its own executor is allowed) and nothing can tell whether it is:

1. Mark the session shut down, so deferred-task wrappers and late work skip the VM.
2. Once the session's executor has drained (stopped, with its last closure returned:
   `closure-actor.md`, Delivery), cancel all active timers, clear the timer registry and
   close the Lua VM. Timers and the VM are touched only on the executor, so after the
   drain nothing else can be using them. The session's own teardown
   stops the executor right after, so this follows promptly. If the executor has not
   drained after a timeout, that is logged once and the teardown keeps waiting: a wedged
   task leaks its VM rather than having it closed under it.
3. Work that reaches the executor after it has stopped does not run; it is work for a
   destroyed session (`closure-actor.md`).

The Server still must not hold its sessions lock while shutting a session down.
`Server.Shutdown` snapshots the session set under the lock, releases it, then
shuts each session down. Otherwise an in-flight deferred task that calls back
into the Server (for example, the app-loader resolving a session) would block
on the held lock and deadlock the join.

## Constraints

- Timer functions can be called from any Lua code running inside the session
- Scheduled functions must not assume they run during the same batch
- If the session is destroyed before a timer fires, the timer is silently cancelled
- `setInterval` timers that are not cleared will run until session shutdown
