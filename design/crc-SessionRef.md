# SessionRef
**Requirements:** R153, R154, R156-165, R167, R174, R192, R229, R236, R237, R238, R255

A handle to one session, plus the per-system rigs reached through it. This is the seam
the coherency carve's steps move session state behind. Every rig reaches the session's
parts through the Session itself: its executor and its Lua session (R229).

## Knows
- srv: the `*Server` that issued it, whose session registry it resolves through (R229)
- vendedID: the session's vended ID (R154)
- lua: a `weak.Pointer` to the `LuaSession` it was issued for. Weak, so the reference
  never keeps the session alive (R156). It is also the identity check: vended IDs are
  reused (the counter resets to 1 when the last session is destroyed), so the ID alone
  could resolve to a newer session (R238).

## Does
- resolve: the live parts, or nothing. The weak pointer's value must be non-nil, must be
  the LuaSession that `Server.luaSessionFor(vendedID)` finds (the Lua session of the
  Session now registered under that vended ID), and must not be shut down; the internal
  ID is that Session's ID (R236, R238).
  Every operation below goes through resolve.
- ID: returns vendedID (R154).
- Valid: resolve succeeds (R238).
- Lua / Browser / Exec: return a rig holding a copy of the reference; no work (R157,
  R165).
- Station: a struct holding all three rigs (R157).
- No rig has a method that waits for the executor: posting returns at once, and
  synchronous execution stays on Server.ExecuteInSession (R237).

### Rigs
Each rig is a struct holding the reference. Every rig has `Ref()` and `Station()`
(R158).
- ExecRig.Post(fn): resolve, then `WebSocketEndpoint.ExecuteInSessionPassive(internalID,
  ...)` with a closure that runs `fn` through `LuaSession.RunInSession(vendedID, ...)`
  so the Lua context is set: in place, since that closure is already on the session
  executor (R255). Returns false without posting when resolve fails (R159,
  R161).
- ExecRig.PostUpdate(fn): the same through `WebSocketEndpoint.ExecuteInSessionAsync`,
  which runs change detection afterward when a browser is connected (R160, R161).
- BrowserRig.Connected: `WebSocketEndpoint.HasConnectionsForSession(internalID)` (R162).
- BrowserRig.Update: `WebSocketEndpoint.RunAfterBatch(internalID)`; executor only
  (R163).
- LuaRig.Session: the resolved `LuaSession`, or nil when invalid; executor only (R164).
- ExecRig.Tailor(name, deadline): a new Tailor on this session (R174).
- ExecRig.PendingTailors: snapshots of the session's open Tailors (R192).

There is no current-session accessor; hosts supply references (R167).

## Collaborators
- Server: issues references (`SessionRef`); luaSessionFor finds the session's Lua session
- WebSocketEndpoint: the session executor, change detection, connection state
- LuaSession: Lua context for posted work; shutdown state for validity
- SessionManager: the session registry, through Server.luaSessionFor

## Sequences
- seq-session-ref.md
