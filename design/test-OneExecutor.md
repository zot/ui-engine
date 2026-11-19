# Test Design: One executor per session
**Source:** crc-LuaSession.md, crc-Server.md, crc-SessionRef.md, crc-LuaHotLoader.md

Until Item 14 each session ran two goroutines that could both be inside its Lua VM:
change detection on the session executor, and anything posted straight to the
LuaSession's own executor (flib's `WithLua`, the hot-loader). A probe on 2026-11-19 caught
them overlapping about half the time and gopher-lua aborting with `fatal error: concurrent
map read and map write`. These tests count overlaps rather than wait for that crash, since
a fatal error ends the test binary instead of failing a test.

## Test: change detection never overlaps work posted from outside
**Purpose:** with a browser connected, change detection (session executor) and Lua work posted from outside (the `WithLua` path) never run at once (R254, R256)
**Input:** a session whose `main.lua` makes variable 1 an `App` and binds variable 2 to the method path `probe()`, which calls a Go recorder; a connection attached so change detection runs; one goroutine posting 100 `PostUpdate` no-ops while another makes 100 `LuaSession.ExecuteInSession` calls whose closures mark themselves in flight for 2 ms and only read Lua state (the path still writes the session global)
**Expected:** the recorder ran more than 0 times (the probe is live) and never while an outside closure was in flight
**Refs:** crc-LuaSession.md, seq-lua-execute.md
**Code:** internal/server/one_executor_test.go
**Alarm:** 1
**Fire alarm:** make LuaSession's execute call fn directly on the caller's goroutine instead of posting it. Red either way: the recorder counts overlaps, or, as on 2026-11-19, the binary aborts with `fatal error: concurrent map writes` first, because both paths set the `session` global through RunInSession. The outside closures only read, but that write is in the path itself
**Inject:** internal/lua/runtime.go:LuaSession.execute
@alarm-hash: 781f549fe38f940d
**Pulled:** 2026-11-19 — rang: the binary aborted with `fatal error: concurrent map writes` (execute bypassed, both paths setting the session global at once)

## Test: hot reload posts its Lua work through the executor
**Purpose:** a reload compiles off the executor and runs the compiled chunk as one piece of posted work (R256)
**Input:** a standalone LuaSession with a recording executor installed by SetExecutor, a file already loaded, and a reload of new content that sets a global
**Expected:** the recording executor ran exactly one piece of work, and the global holds the new value
**Refs:** crc-LuaHotLoader.md, seq-lua-hotload.md
**Code:** internal/lua/hotloader_test.go
**Alarm:** 2
**Fire alarm:** run the compiled chunk with RunCompiled directly instead of through run; the recorder sees no work while the global still changes
**Inject:** internal/lua/hotloader.go:HotLoader.reloadInSession
@alarm-hash: 4f67f208d6b8fcf8
**Pulled:** 2026-11-19 — rang: `reload posted 0 pieces of work to the executor, want 1`

## Test: a file that does not compile never touches the session
**Purpose:** compile errors are found off the executor, so nothing is posted (R257)
**Input:** the same session and recording executor; a reload of content with a syntax error
**Expected:** the reload reports an error, the recording executor ran nothing, and the previous global value is unchanged
**Refs:** crc-LuaHotLoader.md
**Code:** internal/lua/hotloader_test.go
**Alarm:** 3
**Fire alarm:** ignore the compile error in reloadFile and reload anyway; the recorder sees one piece of work for a file that never compiled
**Inject:** internal/lua/hotloader.go:HotLoader.reloadFile
@alarm-hash: 36728a718657ffb5
**Pulled:** 2026-11-19 — rang: `a file that does not compile posted 1 pieces of work, want 0`

## Test: work already on the executor reaches Lua in place
**Purpose:** rigs, `Server.ExecuteInSession` and timers run their Lua without posting to the executor they are on (R255)
**Input:** a session; `ExecRig.Post`, `ExecRig.PostUpdate`, `Server.ExecuteInSession` and a Lua `session:setImmediate`, each setting a global
**Expected:** all four globals are set within 2 seconds
**Refs:** crc-SessionRef.md, crc-Server.md
**Code:** internal/server/one_executor_test.go
**Alarm:** 4
**Fire alarm:** make RunInSession go through run instead of running in place; the work posts to the executor it is already on, waits for itself, and the test times out at 2 seconds
**Inject:** internal/lua/runtime.go:LuaSession.RunInSession
@alarm-hash: 335f4efcdc976a50
**Pulled:** 2026-11-19 — rang: `viaPost never ran within 2 seconds: work on the executor waited for itself`

## Test: a session destroyed from inside its executor closes its VM only after
**Purpose:** Shutdown returns at once and closes the VM after the executor drains, never under the closure that destroyed it (R260, R262)
**Input:** through `ExecRig.Post`, a closure that destroys its own session, then sleeps 50 ms and reads a Lua global, recovering from any panic and recording whether the read worked
**Expected:** the read worked; within 2 seconds the LuaSession reports its VM closed
**Refs:** crc-LuaSession.md, seq-session-timer.md
**Code:** internal/server/one_executor_test.go
**Alarm:** 5
**Fire alarm:** close the VM in Shutdown at once instead of through closeWhenDrained; the read after the destroy panics on the closed VM, and the recorded result says so
**Inject:** internal/lua/runtime.go:LuaSession.Shutdown
@alarm-hash: 4f3cd930a7e8885d
**Pulled:** 2026-11-19 — rang: `the VM was closed under the closure that destroyed its own session`

## Tests removed
`TestShutdownJoinsInFlightExecutor` (R115) and `TestExecutorSkipsQueuedWorkAfterShutdown`
(R116) in `internal/lua/shutdown_race_test.go` test the retired Lua goroutine.
`TestDeferredWrapperSkipsAfterShutdown` (R117) stays, rewritten to mark the session shut
down through Shutdown instead of closing its channels by hand.

## Not tested, deliberately
**R258 and R259** (`main.lua` compiled, and log files opened, off the executor). Both are
the order of two calls in one function, checked by reading; a test would have to stub file
opening and the compiler to observe it.

**R261's timeout log.** It fires only for a closure that never returns, which a test would
have to leak.
