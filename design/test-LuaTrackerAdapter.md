# Test Design: LuaTrackerAdapter
**Source:** crc-LuaTrackerAdapter.md

Until 2026-11-19 the adapter found a variable's session through `varToSession`, a
server-wide index from variable ID to session that was only ever written by
`CreatePathVariable`, which nothing called. The index was always empty, so the store's
`Get`, `GetProperty` and `Destroy` always missed, and with them the Lua
`session:destroyVariable(id)`. It could not have worked filled either: variable IDs repeat
across sessions. (Lua `Variable:getValue()` and `getProperty()` would also reach the
store, but they live in `lib/lua/session.lua`, which sessions do not load; gap O10.)
These tests use two sessions wherever an ID could be confused. Child variables are made
directly in each session's tracker, because creating one through the store panics (gap O9).

## Test: a variable ID resolves in the session that names it
**Purpose:** two sessions each have a variable `-1`; the store reads each in its own session (R241)
**Input:** two sessions with a root variable; in each tracker, a child `-1` with a different property; read `Get` and `GetProperty` for `-1` naming each session
**Expected:** each read returns its own session's value and property
**Refs:** crc-LuaTrackerAdapter.md
**Code:** internal/server/store_adapter_test.go
**Alarm:** 1
**Fire alarm:** make backend ignore its argument and always look up session "1"; the second session's read returns the first's value
**Inject:** internal/server/server.go:luaTrackerAdapter.backend
@alarm-hash: e792e3e61cd2dcef
**Pulled:** 2026-11-19 — rang, in setup rather than at the read: backend always looks up session "1", so the second session's root went into the first's tracker; "root variable for session 2: id 3"

## Test: server variable IDs are counted per session
**Purpose:** each session numbers its server variables from `-1` independently (R240, R45)
**Input:** two sessions; ask the first session's backend for its next server ID, then the second's, then the first's again
**Expected:** IDs `-1`, `-1`, `-2`
**Refs:** crc-LuaBackend.md, crc-LuaTrackerAdapter.md
**Code:** internal/server/store_adapter_test.go
**Alarm:** 2
**Fire alarm:** make NextServerVarID decrement a package-level counter instead of the backend's field; the second session gets `-2`
**Inject:** internal/backend/lua.go:LuaBackend.NextServerVarID
@alarm-hash: aae32f8436d84fe0
**Pulled:** 2026-11-19 — rang: NextServerVarID on a package-level counter; "server variable IDs = [-1 -2 -3], want [-1 -1 -2]"

## Test: session:destroyVariable destroys the variable
**Purpose:** the Lua call reaches the store with its own session and the variable is gone (R242, R243)
**Input:** a session with a root variable and a child `-1` in its tracker; Lua calls `session:destroyVariable(-1)`
**Expected:** the session's tracker no longer has the variable
**Refs:** crc-LuaSession.md, crc-LuaTrackerAdapter.md
**Code:** internal/server/store_adapter_test.go
**Alarm:** 3
**Fire alarm:** in the Go destroyVariable override, pass an empty session ID to the store; the variable survives
**Inject:** internal/lua/runtime.go:LuaSession.addGoSessionMethods
@alarm-hash: cc0ecc77b8e03375
**Pulled:** 2026-11-19 — rang: destroyVariable override passes an empty session ID; "variable -1 survived session:destroyVariable (R243)"

## Test: the adapter keeps no copy of the backend
**Purpose:** the adapter reads whatever backend the session holds now (R239)
**Input:** a session; replace its backend on the Session with a fresh LuaBackend; ask the adapter for the session's tracker
**Expected:** the fresh backend's tracker
**Refs:** crc-LuaTrackerAdapter.md
**Code:** internal/server/store_adapter_test.go
