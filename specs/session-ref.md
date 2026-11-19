# Session References and Rigs

**Language:** Go
**Environment:** ui-engine backend (`internal/server`, re-exported through `cli`)

## Purpose

Give Go code one handle to a session from which it can reach everything the session
owns, instead of carrying session IDs and looking each part up in a different map.

A session's state was spread across maps owned by several objects (`SessionManager`,
`WebSocketEndpoint`, `Server`, the store adapter, the variable store), keyed by two kinds
of session ID, and every operation reassembled the session from those maps. A session
reference replaces that with navigation: hold the reference, ask it for the part you need.

This is a piece-at-a-time restructuring. The references were first implemented over the
existing maps, so nothing that existed changed. Each later step moves one part into the
session itself, behind the same references, without changing callers; "Where a session's
parts live" below says which parts have moved.

## Concepts

- **Session reference** (`SessionRef`): a small value naming one session. It is cheap to
  copy and safe to hold after the session ends. It never keeps a session alive.
- **Rig**: a value obtained from a reference that carries one system's operations on
  that session. There is one rig per system: Lua, browser, and execution.
- **Station**: a struct holding every rig of one session. Any rig can produce it, so
  from any rig every other system is one step away.

A session reference is valid while its Lua session is running. After the session is
destroyed, the reference is invalid, and operations through it report that rather than
failing in other ways. Vended IDs are reused once every session is gone, so a reference
also becomes invalid, rather than following the ID, when a new session takes its ID.

## API

```go
// Obtaining a reference
func (s *Server) SessionRef(vendedID string) (SessionRef, bool) // false if no such session

type SessionRef struct { /* unexported */ }

func (r SessionRef) ID() string          // the vended session ID
func (r SessionRef) Valid() bool         // true while the session's Lua session is running
func (r SessionRef) Lua() LuaRig
func (r SessionRef) Browser() BrowserRig
func (r SessionRef) Exec() ExecRig
func (r SessionRef) Station() Station

type Station struct {
    Lua     LuaRig
    Browser BrowserRig
    Exec    ExecRig
}

// Every rig can return to its reference and produce the station.
func (v LuaRig) Ref() SessionRef
func (v LuaRig) Station() Station
// (likewise for BrowserRig and ExecRig)

// Execution: post work to the session's executor without waiting.
func (v ExecRig) Post(fn func()) bool       // passive: no change detection afterward
func (v ExecRig) PostUpdate(fn func()) bool // active: change detection and push afterward

// Browser: whether a browser is attached, and pushing changes to it.
func (v BrowserRig) Connected() bool
func (v BrowserRig) Update() // run change detection and push now; call only on the executor

// Lua: the session's Lua state, usable only on the session's executor.
func (v LuaRig) Session() *lua.LuaSession
```

- `Post` and `PostUpdate` return at once. They return false, and do nothing, when the
  reference is invalid.
- Work posted through `Post` or `PostUpdate` runs on the session's executor with the
  session's Lua context set, the same context `ExecuteInSession` provides.
- `PostUpdate` runs change detection after the work, and pushes changes when a browser is
  connected. `Post` does neither. With no browser connected, posted work still runs.
- `BrowserRig.Update` lets passive work trigger change detection itself when it decides
  to, for example after an error.
- Rigs and the station are values. Obtaining them does no work beyond copying the
  reference.

## Dangerous execution

A synchronous `ExecuteInSession` waits for the session's executor. Called from inside an
executor it waits on the queue it is running in, and deadlocks. Called from any other
goroutine it is safe, which is true of every caller today, but nothing in the call shows
which kind it is. So synchronous execution is marked as dangerous and made loud, and
callers move to session references (or are reviewed as safe) a site at a time.

```go
func (s *Server) PrimitiveExecuteInSession(vendedID string, fn func() (interface{}, error), dangerous bool) (interface{}, error)
```

- `Server.ExecuteInSession` keeps its signature and delegates all its behaviour to
  `PrimitiveExecuteInSession` with `dangerous` set to `true`.
- With `dangerous` false, `PrimitiveExecuteInSession` behaves exactly as
  `ExecuteInSession` did before this change.
- With `dangerous` true, what happens depends on the session config property
  `AllowDangerousExecution` (`allow_dangerous_execution` in the config file):
  - **set:** log a warning naming the site that called `ExecuteInSession` (file and
    line), once per site for the life of the process, then proceed as if `dangerous` were
    false;
  - **not set:** panic, naming the same site, before queueing anything.
- `AllowDangerousExecution` defaults to **not set**, so a dangerous call panics. It
  defaulted to set while callers were moved, with the warnings as the list of remaining
  callers; once none warned in ui-engine, Frictionless or ark, the default flipped. An
  embedder with a caller not yet moved sets it to get the warning instead of the panic.
- **ui-engine's own code makes no dangerous call.** Work that needs no result, only
  change detection afterward, posts with the session's `ExecRig.PostUpdate` and does not
  wait: pushing changed viewdefs to a session, and refreshing a session after a hot
  reload.

## Where a session's parts live

Each part has exactly one home. When a part moves into the session, the old way of
reaching it keeps working by reading the new home; it never keeps a copy.

- **The executor** belongs to the session (see `closure-actor.md`).
- **The Lua session** belongs to the session. It is attached while the session is
  created, before `main.lua` runs, so the session can be found by its vended ID while it
  starts up, and it does not change for the life of the session. No other table holds it:
  not the server, and not the variable store's adapter.
- **Finding a session's Lua session by vended ID** goes from the vended ID to the session
  and asks it. `Server.GetLuaSession` keeps its signature and works this way, as do
  browser messages, change detection and the debug page.
- **Every running session** is enumerated from the session registry. Hot-loading and
  viewdef pushes reach every session this way.
- **Server shutdown** takes the list of sessions, lets go of the registry, and only then
  shuts each Lua session down, so a task that reaches back into the server while a session
  shuts down cannot deadlock it.
- **A destroyed session cannot be found by its vended ID** from the moment its IDs are
  dropped, including while its Lua session is being torn down.
- **The backend** belongs to the session. The variable store's adapter reaches a
  session's backend through the session registry and keeps no copy.
- **Each session numbers its own server-created variables** (negative IDs from `-1`, see
  `protocol.md`), so the counter belongs to that session's backend, beside the variables
  it numbers.
- **A variable ID names a variable only within its session**: every session has a
  variable 1 and a variable `-1`. So every variable operation names its session
  (creating a variable, reading its value or a property, updating it, destroying it), and
  nothing maps a variable ID back to a session. A Lua session names its own. Through this,
  `session:destroyVariable(id)` destroys that session's variable (`libraries.md`).
- **A browser connection holds its session.** When a browser connects to a session that
  exists, the connection keeps the session itself, and the session records the
  connection; on disconnect both are undone together. Each direction of that relationship
  is kept once. Messages from the connection go to its session's executor; a connection
  whose session does not exist reaches nothing.
- **A session's connections are the session's own.** Whether a browser is connected to a
  session, and which connections a broadcast to it reaches, come from the session. A
  destroyed session has no connections, even if a browser that was attached to it has not
  yet closed: change detection does not run for it, and a broadcast to it sends nothing.
- **Reconnection needs no token.** A browser can rejoin any session that has not timed
  out (`interfaces.md`); nothing issues or checks a reconnect token.
- **A session keeps no URL paths yet.** Registering presenters at URL paths after the
  session ID (`interfaces.md`, "SPA History Management") is not built: nothing lets a
  backend register one, and the server hands every path under a session to the browser.
  Until it is built, no table of a session's URL paths exists on the server.

Execution through the server reads the same home as the rigs:

- `Server.ExecuteInSessionAsync` is `ExecRig.PostUpdate` on a reference to the session,
  and does nothing when there is no such session.
- `Server.ExecuteInSession` finds the session the same way a reference does. It stays off
  the rigs: no rig waits for the executor, because a wait made from inside an executor
  deadlocks (see "Dangerous execution").

## Non-goals

- No notion of a "current session". ui-engine serves many sessions; a host that has a
  current one (Frictionless's MCP server does) supplies the reference itself.
- No synchronous method on any rig.
