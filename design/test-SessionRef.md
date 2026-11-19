# Test Design: Session references and dangerous execution
**Source:** crc-SessionRef.md

Most of these properties are invisible when violated: a reference that resolves to the
wrong session still runs the work, a passive post that runs change detection only costs
time, and a warning repeated per call only adds noise. Each gets an alarm.

## Test: a reference resolves only its own session
**Purpose:** Valid is false after the session is destroyed, including when a new session reuses its vended ID (R155, R156)
**Input:** create session "1", take a ref; destroy it (the vended counter resets); create a new session, which is "1" again
**Expected:** the old ref is invalid and its Post returns false; a fresh ref for "1" is valid
**Refs:** crc-SessionRef.md, seq-session-ref.md
**Code:** internal/server/sessionref_test.go
**Alarm:** 1
**Fire alarm:** drop both of resolve's session checks (identity and shutdown), keeping only that `luaSessions[vendedID]` exists; the old ref reports valid against the new session. Dropping either check alone does not ring here, because destroy both unregisters and shuts down; each check has its own test below
**Inject:** internal/server/sessionref.go:SessionRef.resolve
@alarm-hash: 741bf325fdc32656
**Pulled:** 2026-11-18 — 2026-11-18 — rang: dropped both resolve checks; "old ref reports valid against the new session 1". Dropping identity alone did not ring (shutdown also catches it), which is why #6 and #7 exist

## Test: Post runs with the Lua context and without change detection
**Purpose:** Post runs fn on the executor with the session context and runs no afterBatch; PostUpdate runs afterBatch (R159, R160)
**Input:** a server with a counting afterBatch callback and one connected session; Post a fn that records the `session` global; then PostUpdate a no-op; wait for both on a channel
**Expected:** fn saw the session table; afterBatch count is 0 after Post and 1 after PostUpdate
**Refs:** crc-SessionRef.md, crc-WebSocketEndpoint.md
**Code:** internal/server/sessionref_test.go
**Alarm:** 2
**Fire alarm:** implement ExecuteInSessionPassive by calling ExecuteInSessionAsync; the afterBatch count after Post is 1
**Inject:** internal/server/websocket.go:WebSocketEndpoint.ExecuteInSessionPassive
@alarm-hash: be2074a243ad0d8e
**Pulled:** 2026-11-19 — re-pulled after the executor moved to Session: ExecuteInSessionPassive implemented through ExecuteInSessionAsync; "afterBatch ran 2 times after Post, want 0" *Earlier —* 2026-11-18 — 2026-11-18 — rang: ExecuteInSessionPassive routed through ExecuteInSessionAsync; "afterBatch ran 2 times after Post, want 0"
## Test: posting on an invalid reference does nothing
**Purpose:** Post and PostUpdate return false and queue nothing when the reference is invalid (R161)
**Input:** a destroyed session's ref; Post a fn that closes a channel
**Expected:** Post returns false; the channel stays open after a short wait
**Refs:** crc-SessionRef.md
**Code:** internal/server/sessionref_test.go
**Alarm:** 3
**Fire alarm:** remove the resolve check in Post so it posts by vended ID; Post returns true
**Inject:** internal/server/sessionref.go:ExecRig.Post
@alarm-hash: 6147ef2da912ff9d
**Pulled:** 2026-11-19 — re-pulled after the rename to ExecRig: Post's resolve check removed; "Post on a destroyed session returned true" *Earlier —* 2026-11-18 — 2026-11-18 — rang: Post returned true on a failed resolve; "Post on a destroyed session returned true"
## Test: rigs and station round-trip
**Purpose:** every rig returns its reference and the same station (R157, R158)
**Input:** a valid ref
**Expected:** `ref.Lua().Ref().ID() == ref.ID()`, and likewise for Browser and Exec; `ref.Exec().Station().Lua.Ref().ID() == ref.ID()`
**Refs:** crc-SessionRef.md
**Code:** internal/server/sessionref_test.go

## Test: a dangerous call warns once per site
**Purpose:** with AllowDangerousExecution set, ExecuteInSession logs one warning per calling site and still runs (R168, R170)
**Input:** a capturing logger; call ExecuteInSession three times from one line and once from another
**Expected:** exactly two warnings, each naming its own file and line; all four calls return fn's result
**Refs:** crc-Server.md, seq-session-ref.md
**Code:** internal/server/dangerous_test.go
**Alarm:** 4
**Fire alarm:** skip recording the site in the seen set; four warnings appear
**Inject:** internal/server/dangerous.go:Server.checkDangerous
@alarm-hash: a2580e2c07a5fccc
**Pulled:** 2026-11-18 — 2026-11-18 — rang: site not recorded in the seen set; "got 4 warnings, want 2 (one per site)"

## Test: a dangerous call panics when not allowed
**Purpose:** with AllowDangerousExecution unset, ExecuteInSession panics naming the site and queues nothing (R171)
**Input:** config with the property unset; ExecuteInSession with a fn that closes a channel, under recover
**Expected:** a panic whose message names this test file; the channel stays open
**Refs:** crc-Server.md
**Code:** internal/server/dangerous_test.go
**Alarm:** 5
**Fire alarm:** move the panic after the queueing call; the channel closes before the panic
**Inject:** internal/server/dangerous.go:Server.PrimitiveExecuteInSession
@alarm-hash: a1f0e82f2a64feee
**Pulled:** 2026-11-18 — 2026-11-18 — rang: checkDangerous moved after executeInSession; "the dangerous call ran its work before panicking"

## Test: the non-dangerous path is silent
**Purpose:** PrimitiveExecuteInSession with dangerous false neither warns nor panics, with the property unset (R169)
**Input:** property unset, capturing logger; call the primitive with false
**Expected:** fn's result returned, no warning, no panic
**Refs:** crc-Server.md
**Code:** internal/server/dangerous_test.go

## Test: AllowDangerousExecution defaults to set
**Purpose:** the default config allows dangerous execution (R172)
**Input:** `config.DefaultConfig()`
**Expected:** `Session.AllowDangerousExecution` is true
**Refs:** crc-Config.md
**Code:** internal/server/dangerous_test.go

## Test: identity alone invalidates a replaced session
**Purpose:** a reference is invalid once another Lua session is registered under its vended ID, even while its own Lua session still runs (R155)
**Input:** two sessions; take a ref to the first; register the second's Lua session under the first's vended ID without shutting the first down
**Expected:** the ref is invalid
**Refs:** crc-SessionRef.md
**Code:** internal/server/sessionref_test.go
**Alarm:** 6
**Fire alarm:** in resolve, compare only that `luaSessions[vendedID]` is non-nil instead of that it is this ref's Lua session; the ref reports valid
**Inject:** internal/server/sessionref.go:SessionRef.resolve
@alarm-hash: 741bf325fdc32656
**Pulled:** 2026-11-18 — 2026-11-18 — rang: identity check reduced to non-nil; "ref still valid after another Lua session took its vended ID"

## Test: shutdown alone invalidates a reference
**Purpose:** a reference is invalid once its Lua session has shut down, even while it is still registered, as during a server shutdown (R155)
**Input:** one session and a ref; call the Lua session's Shutdown without unregistering it
**Expected:** the ref is invalid
**Refs:** crc-SessionRef.md, crc-LuaSession.md
**Code:** internal/server/sessionref_test.go
**Alarm:** 7
**Fire alarm:** drop the IsShutdown check from resolve; the ref reports valid
**Inject:** internal/server/sessionref.go:SessionRef.resolve
@alarm-hash: 741bf325fdc32656
**Pulled:** 2026-11-18 — 2026-11-18 — rang: IsShutdown check dropped; "ref still valid after its Lua session shut down"

## Test: the browser rig reports connection and pushes on demand
**Purpose:** Connected follows browser attachment, and Update called on the executor runs change detection once (R162, R163)
**Input:** a counting afterBatch; Connected before and after attaching a browser; a passive Post that calls `Browser().Update()`
**Expected:** Connected false then true; afterBatch count 1 after the post
**Refs:** crc-SessionRef.md, crc-WebSocketEndpoint.md
**Code:** internal/server/sessionref_test.go
**Alarm:** 8
**Fire alarm:** make RunAfterBatch skip the afterBatch call; the count stays 0
**Inject:** internal/server/websocket.go:WebSocketEndpoint.RunAfterBatch
@alarm-hash: 5cf1a9b417f1c466
**Pulled:** 2026-11-18 — 2026-11-18 — rang: RunAfterBatch skipped afterBatch; "afterBatch ran 0 times after Update in a passive post, want 1"
