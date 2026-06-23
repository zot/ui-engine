# Session Timers — setImmediate, setTimeout, setInterval

**Language:** Go, Lua
**Environment:** ui-engine backend (internal/lua, internal/server)

## Purpose

Allow Lua code to schedule functions for deferred or timed execution. This avoids the deadlock that occurs when Lua code (running on the executor goroutine inside a ChanSvc turn) tries to call `ExecuteInSession` synchronously.

## Problem

The session execution model has two nested serialization layers (closure-actor pattern):

1. **ChanSvc** — serializes all session operations (WebSocket messages, external calls)
2. **executorChan** — serializes Lua VM access within a ChanSvc turn

When Lua code is running, both layers are occupied. A synchronous `ExecuteInSession` call from Lua would try to queue onto ChanSvc, but ChanSvc is blocked waiting for the executor — deadlock.

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
- Runs `fn` through `luaSession.ExecuteInSession` (sets session context, uses executor)
- Triggers `afterBatch` after execution
- Errors are logged, not returned (fire-and-forget)

## Mechanism

All three timer functions share the same execution path:

1. Capture the Lua function and the current `tracker.ComputingVar`
2. Allocate a handle from an incrementing counter; store in a registry with a `cancelled` flag
3. Schedule via the `onDefer` callback (set by Server), which calls `Server.ExecuteInSessionAsync`
4. Before executing, check the `cancelled` flag — skip if cancelled
5. The function runs through ChanSvc → executor with session context, then `afterBatch` pushes changes

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

A session's Lua VM runs on a single executor goroutine. `Shutdown` must free
that VM (`State.Close()`) with no work touching it concurrently, or the process
crashes with a nil-pointer dereference inside the Lua VM. Deferred tasks
(timers, and the app-loader a timer may invoke) are the work most likely to be
in flight when a session shuts down.

Shutdown proceeds in order:

1. Signal shutdown by closing the session's `done` channel.
2. Wait for the executor goroutine to finish any in-flight work item and
   return, bounded by a timeout. On timeout, log and proceed: a wedged task
   must not hang `ark stop` forever.
3. Cancel all active timers and clear the timer registry.
4. Close the Lua VM.

The executor cooperates with the signal in two places:

- When it dequeues a work item, it re-checks `done` before running it. If
  shutdown has begun it returns a shutdown error to the caller instead of
  executing, so work queued during teardown never runs.
- Each deferred-task wrapper re-checks `done` before touching the VM, skipping
  execution when shutdown has begun.

Because the executor join in step 2 can block while an in-flight task runs, the
Server must not hold its sessions lock while shutting a session down.
`Server.Shutdown` snapshots the session set under the lock, releases it, then
shuts each session down. Otherwise an in-flight deferred task that calls back
into the Server (for example, the app-loader resolving a session) would block
on the held lock and deadlock the join.

## Constraints

- Timer functions can be called from any Lua code running inside the session
- Scheduled functions must not assume they run during the same batch
- If the session is destroyed before a timer fires, the timer is silently cancelled
- `setInterval` timers that are not cleared will run until session shutdown
