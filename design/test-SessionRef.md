# Test Design: Session references and dangerous execution
**Source:** crc-SessionRef.md

Most of these properties are invisible when violated: a reference that resolves to the
wrong session still runs the work, a passive post that runs change detection only costs
time, and a warning repeated per call only adds noise. Each gets an alarm.

## Test: a reference resolves only its own session
**Purpose:** Valid is false after the session is destroyed, including when a new session reuses its vended ID (R238, R156)
**Input:** create session "1", take a ref; destroy it (the vended counter resets); create a new session, which is "1" again
**Expected:** the old ref is invalid and its Post returns false; a fresh ref for "1" is valid
**Refs:** crc-SessionRef.md, seq-session-ref.md
**Code:** internal/server/sessionref_test.go
**Alarm:** 1
**Fire alarm:** drop resolve's identity check, keeping only that `luaSessionFor(vendedID)` finds a Lua session; after the vended ID is reused the old ref reports valid against the new session
**Inject:** internal/server/sessionref.go:SessionRef.resolve
@alarm-hash: 573a000e40790dda
**Pulled:** 2026-11-19 — re-pulled after resolve moved onto luaSessionFor: identity check dropped; "old ref reports valid against the new session 1" *Earlier —* 2026-11-18 — 2026-11-18 — rang: dropped both resolve checks; "old ref reports valid against the new session 1". Dropping identity alone did not ring (shutdown also catches it), which is why #6 and #7 exist
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
**Purpose:** a reference is invalid once its vended ID leads to another session, even while its own Lua session still runs (R238)
**Input:** two sessions; take a ref to the first; point the first's vended ID at the second session in the registry, without shutting the first down
**Expected:** the ref is invalid
**Refs:** crc-SessionRef.md
**Code:** internal/server/sessionref_test.go
**Alarm:** 6
**Fire alarm:** in resolve, check only that `luaSessionFor(vendedID)` is non-nil instead of that it is this ref's Lua session; the ref reports valid
**Inject:** internal/server/sessionref.go:SessionRef.resolve
@alarm-hash: 573a000e40790dda
**Pulled:** 2026-11-19 — re-pulled after resolve moved onto luaSessionFor: identity check reduced to non-nil; "ref still valid after another Lua session took its vended ID" *Earlier —* 2026-11-18 — 2026-11-18 — rang: identity check reduced to non-nil; "ref still valid after another Lua session took its vended ID"
## Test: shutdown alone invalidates a reference
**Purpose:** a reference is invalid once its Lua session has shut down, even while it is still registered, as during a server shutdown (R238)
**Input:** one session and a ref; call the Lua session's Shutdown without unregistering it
**Expected:** the ref is invalid
**Refs:** crc-SessionRef.md, crc-LuaSession.md
**Code:** internal/server/sessionref_test.go
**Alarm:** 7
**Fire alarm:** drop the shut-down check from luaSessionFor, which is the only one since resolve checks identity alone; the ref reports valid
**Inject:** internal/server/server.go:Server.luaSessionFor
@alarm-hash: c8bf0cc912e0a61a
**Pulled:** 2026-11-19 — moved to luaSessionFor: its shut-down check dropped; "ref still valid after its Lua session shut down" *Earlier —* 2026-11-18 — 2026-11-18 — rang: IsShutdown check dropped; "ref still valid after its Lua session shut down"
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

## Test: the Lua session lives on the session
**Purpose:** the session holds its Lua session, and lookups by vended ID return that one (R230, R231, R232)
**Input:** a server and one session
**Expected:** `GetLuaSession(vendedID)` is non-nil and is the session's `LuaSession()`; `SessionRef(vendedID)` is valid
**Refs:** crc-Session.md, crc-Server.md
**Code:** internal/server/session_lua_test.go

## Test: a session can be found while main.lua runs
**Purpose:** the Lua session is attached before main.lua runs, so work main.lua defers reaches it (R230)
**Input:** a main.lua that calls `session:setImmediate` with a function that records it ran; create a session
**Expected:** the recorded flag is set within 2 seconds
**Refs:** crc-Server.md, seq-session-create-backend.md
**Code:** internal/server/session_lua_test.go
**Alarm:** 9
**Fire alarm:** attach the Lua session after CreateLuaSession instead of before; the deferred function is dropped because its session cannot be found
**Inject:** internal/server/server.go:Server.CreateLuaBackendForSession
@alarm-hash: 4751fd84660fbdf7
**Pulled:** 2026-11-19 — re-pulled after 9a removed the store adapter's SetBackend from this function: Lua session attached after CreateLuaSession; "work main.lua deferred never ran" *Earlier —* 2026-11-19 — rang: Lua session attached after CreateLuaSession; "work main.lua deferred never ran: its session could not be found while main.lua ran"
## Test: a second attach panics
**Purpose:** a session's Lua session never changes (R230)
**Input:** a session with a Lua session attached; attach another
**Expected:** the second attach panics
**Refs:** crc-Session.md
**Code:** internal/server/session_lua_test.go
**Alarm:** 10
**Fire alarm:** make attachLua overwrite silently; nothing panics
**Inject:** internal/server/session.go:Session.attachLua
@alarm-hash: 43d7da6b3f239a49
**Pulled:** 2026-11-19 — rang: attachLua overwrites with Store; "a second attachLua did not panic"

## Test: a destroyed session cannot be found during teardown
**Purpose:** a session is unreachable by vended ID from the moment its IDs are dropped, while its Lua session is still being torn down (R234)
**Input:** wrap the destroy callback so it records `GetLuaSession(vendedID)` and `SessionRef(vendedID)` before calling `DestroyLuaBackendForSession`; destroy the session
**Expected:** inside the callback, GetLuaSession is nil and SessionRef reports false
**Refs:** crc-SessionManager.md, crc-Server.md
**Code:** internal/server/session_lua_test.go
**Alarm:** 11
**Fire alarm:** in DestroySession, remove the session and its vended ID from the registry after the destroy callback instead of before; GetLuaSession still finds the session inside the callback. Moving only the vended ID mapping does not ring, because GetByVendedID also needs the session entry
**Inject:** internal/server/session_manager.go:SessionManager.DestroySession
@alarm-hash: 7abf44f1ce94e267
**Pulled:** 2026-11-19 — re-pulled after 9c: session and vended ID removed from the registry after the destroy callback; "GetLuaSession(1) found the session during its teardown (R234)" *Earlier —* 2026-11-19 — rang: session and vended ID removed from the registry after the destroy callback; "GetLuaSession(1) found the session during its teardown (R234)"
## Test: nothing is found after server shutdown
**Purpose:** after `Server.Shutdown` every Lua session reports shut down, so lookups find nothing, as they did when the server's map was cleared (R232, R118)
**Input:** a server with two sessions; call Shutdown
**Expected:** GetLuaSession is nil and SessionRef reports false for both vended IDs; GetSessionIDs is empty
**Refs:** crc-Server.md
**Code:** internal/server/session_lua_test.go
**Alarm:** 12
**Fire alarm:** drop the shut-down check from luaSessionFor; GetLuaSession returns the shut-down Lua sessions
**Inject:** internal/server/server.go:Server.luaSessionFor
@alarm-hash: c8bf0cc912e0a61a
**Pulled:** 2026-11-19 — rang: shut-down check dropped from luaSessionFor; "GetLuaSession(1) found a Lua session after Shutdown"

## Test: sessions are enumerated from the registry
**Purpose:** GetSessionIDs reads the session registry (R233)
**Input:** three sessions; then destroy one
**Expected:** GetSessionIDs returns the three vended IDs, then the two that remain
**Refs:** crc-Server.md, crc-SessionManager.md
**Code:** internal/server/session_lua_test.go

## Test: ExecuteInSessionAsync is an active post
**Purpose:** `ExecuteInSessionAsync` runs its work through `ExecRig.PostUpdate`, so change detection follows it, and does nothing for an unknown session (R235)
**Input:** a counting afterBatch and an attached browser; ExecuteInSessionAsync with a function that records it ran; then ExecuteInSessionAsync on an unknown vended ID
**Expected:** the function ran and afterBatch ran once after it; the unknown ID returns without running anything
**Refs:** crc-Server.md, crc-SessionRef.md
**Code:** internal/server/session_lua_test.go
**Alarm:** 13
**Fire alarm:** implement ExecuteInSessionAsync with `Exec().Post` instead of `PostUpdate`; afterBatch runs 0 times
**Inject:** internal/server/server.go:Server.ExecuteInSessionAsync
@alarm-hash: 80df95eb0f75bace
**Pulled:** 2026-11-19 — rang: ExecuteInSessionAsync through Post; "afterBatch ran 0 times after ExecuteInSessionAsync, want 1 (R235)"
