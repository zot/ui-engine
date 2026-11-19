# Test Design: Session Executor
**Source:** crc-SessionManager.md

The executor used to live in a map on WebSocketEndpoint, created on first use and never
stopped, so every destroyed session leaked its goroutine (found 2026-11-18, coherency
carve Item 7). These tests count goroutines, so each records `runtime.NumGoroutine`
before it starts and polls for up to 2 seconds for the count to settle.

## Test: destroying a session ends its executor
**Purpose:** the leak itself: the executor is born with the session and stops with it (R224, R225, R227)
**Input:** record the goroutine count; create 20 sessions through SessionManager, run one closure on each executor, destroy them all
**Expected:** the count returns to the recorded value
**Refs:** crc-SessionManager.md, crc-Session.md, seq-session-executor.md
**Code:** internal/server/session_executor_test.go
**Alarm:** 1
**Fire alarm:** drop the stopExecutor call from DestroySession; the count stays 20 above
**Inject:** internal/server/session_manager.go:SessionManager.DestroySession
@alarm-hash: 7abf44f1ce94e267
**Pulled:** 2026-11-19 — re-pulled after 9c removed urlPaths from DestroySession: stopExecutor dropped; "goroutines: 22, want 2 (leaked 20)" *Earlier —* 2026-11-19 — rang: stopExecutor dropped from DestroySession; "goroutines: 22, want 2 (leaked 20)"
## Test: the destroy callback still has the executor
**Purpose:** the executor stops only after backend teardown, so teardown can still use it (R225)
**Input:** a destroy callback that runs a SvcSync closure on the session's executor and records its result; destroy the session
**Expected:** the closure ran and the SvcSync returned no error
**Refs:** crc-SessionManager.md, seq-session-executor.md
**Code:** internal/server/session_executor_test.go
**Alarm:** 2
**Fire alarm:** in DestroySession, stop the executor before calling the destroy callback; the callback's SvcSync returns `ErrStopped`
**Inject:** internal/server/session_manager.go:SessionManager.DestroySession
@alarm-hash: 7abf44f1ce94e267
**Pulled:** 2026-11-19 — re-pulled after 9c: executor stopped before the destroy callback; "teardown could not use the executor: ran=false err=executor stopped (R225)" *Earlier —* 2026-11-19 — rang: executor stopped before the destroy callback; "teardown could not use the executor: ran=false err=executor stopped (R225)"
## Test: a session destroys itself from inside its executor
**Purpose:** destruction from the session's own executor completes, and the caller waiting on that closure gets its result (R226)
**Input:** through WebSocketEndpoint.ExecuteInSession, run a closure that destroys its own session and returns 7; 2-second deadline
**Expected:** ExecuteInSession returns 7 and no error; the session is gone; the goroutine count returns
**Refs:** crc-SessionManager.md, crc-ChanSvc.md, seq-session-executor.md
**Code:** internal/server/session_executor_test.go

## Test: a failed creation leaks no executor
**Purpose:** when the create callback fails, the new session's executor is stopped (R227)
**Input:** record the goroutine count; a create callback that returns an error; attempt 20 creations
**Expected:** every creation fails; the count returns to the recorded value
**Refs:** crc-SessionManager.md, seq-session-executor.md
**Code:** internal/server/session_executor_test.go
**Alarm:** 3
**Fire alarm:** drop the stopExecutor call from CreateSession's failure path; the count stays 20 above
**Inject:** internal/server/session_manager.go:SessionManager.CreateSession
@alarm-hash: 1230db129da78191
**Pulled:** 2026-11-19 — re-pulled after 9c removed urlPaths from CreateSession's failure path: stopExecutor dropped; "goroutines: 22, want 2 (leaked 20)" *Earlier —* 2026-11-19 — rang: stopExecutor dropped from the CreateSession failure path; "goroutines: 22, want 2 (leaked 20)"
## Test: work for a destroyed session creates nothing and never blocks
**Purpose:** a post for a session that is gone neither blocks nor resurrects an executor (R228)
**Input:** create and destroy a session; record the goroutine count; post to its internal ID 20 times each through ExecuteInSession, ExecuteInSessionAsync and ExecuteInSessionPassive, under a 2-second deadline
**Expected:** every call returns before the deadline; ExecuteInSession returns an error; no closure runs; the count does not rise
**Refs:** crc-WebSocketEndpoint.md, seq-session-executor.md
**Code:** internal/server/session_executor_test.go
**Alarm:** 4
**Fire alarm:** make executor build and start a fresh ChanSvc when the session is missing (the old getOrCreateSvc); the closures run and the count rises
**Inject:** internal/server/websocket.go:WebSocketEndpoint.executor
@alarm-hash: 91d142e928e35794
**Pulled:** 2026-11-19 — rang: executor builds and starts a fresh ChanSvc for a missing session; "0 of 20 returned an error", "60 closures ran for a destroyed session", "leaked 59"
