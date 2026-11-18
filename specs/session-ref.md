# Session References and Views

**Language:** Go
**Environment:** ui-engine backend (`internal/server`, re-exported through `cli`)

## Purpose

Give Go code one handle to a session from which it can reach everything the session
owns, instead of carrying session IDs and looking each part up in a different map.

Today a session's state is spread across maps owned by several objects (`SessionManager`,
`WebSocketEndpoint`, `Server`, the store adapter, the variable store), keyed by two kinds
of session ID. Every operation reassembles the session from those maps. A session
reference replaces that with navigation: hold the reference, ask it for the part you need.

This is the first step of a piece-at-a-time restructuring. The references described here
are implemented over today's maps, so nothing that exists changes. Later steps move each
part into the session itself, behind the same references, without changing callers.

## Concepts

- **Session reference** (`SessionRef`): a small value naming one session. It is cheap to
  copy and safe to hold after the session ends. It never keeps a session alive.
- **View**: a value obtained from a reference that carries one system's operations on
  that session. There is one view per system: Lua, browser, and execution.
- **Station**: a struct holding every view of one session. Any view can produce it, so
  from any view every other system is one step away.

A session reference is valid while its Lua session is running. After the session is
destroyed, or the Lua environment is restarted, the reference is invalid, and operations
through it report that rather than failing in other ways.

## API

```go
// Obtaining a reference
func (s *Server) SessionRef(vendedID string) (SessionRef, bool) // false if no such session

type SessionRef struct { /* unexported */ }

func (r SessionRef) ID() string          // the vended session ID
func (r SessionRef) Valid() bool         // true while the session's Lua session is running
func (r SessionRef) Lua() LuaView
func (r SessionRef) Browser() BrowserView
func (r SessionRef) Exec() ExecView
func (r SessionRef) Station() Station

type Station struct {
    Lua     LuaView
    Browser BrowserView
    Exec    ExecView
}

// Every view can return to its reference and produce the station.
func (v LuaView) Ref() SessionRef
func (v LuaView) Station() Station
// (likewise for BrowserView and ExecView)

// Execution: post work to the session's executor without waiting.
func (v ExecView) Post(fn func()) bool       // passive: no change detection afterward
func (v ExecView) PostUpdate(fn func()) bool // active: change detection and push afterward

// Browser: whether a browser is attached, and pushing changes to it.
func (v BrowserView) Connected() bool
func (v BrowserView) Update() // run change detection and push now; call only on the executor

// Lua: the session's Lua state, usable only on the session's executor.
func (v LuaView) Session() *lua.LuaSession
```

- `Post` and `PostUpdate` return at once. They return false, and do nothing, when the
  reference is invalid.
- Work posted through `Post` or `PostUpdate` runs on the session's executor with the
  session's Lua context set, the same context `ExecuteInSession` provides.
- `PostUpdate` runs change detection after the work, and pushes changes when a browser is
  connected. `Post` does neither. With no browser connected, posted work still runs.
- `BrowserView.Update` lets passive work trigger change detection itself when it decides
  to, for example after an error.
- Views and the station are values. Obtaining them does no work beyond copying the
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
- `AllowDangerousExecution` defaults to set while callers are moved. The warnings are the
  list of remaining callers. When no site warns any more, the default is meant to flip.

## Non-goals

- No notion of a "current session". ui-engine serves many sessions; a host that has a
  current one (Frictionless's MCP server does) supplies the reference itself.
- No other change to existing APIs. `ExecuteInSessionAsync` and the session maps stay as
  they are in this step.
