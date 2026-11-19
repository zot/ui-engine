# SessionRef
**Requirements:** R153-165, R167, R173, R174, R192

A handle to one session, plus the per-system rigs reached through it. This is the seam
the coherency carve's later steps move session state behind; in this step every rig is
implemented over the existing maps on Server and WebSocketEndpoint.

## Knows
- srv: the `*Server` that issued it, the registry it resolves through (R173)
- vendedID: the session's vended ID (R154)
- lua: a `weak.Pointer` to the `LuaSession` it was issued for. Weak, so the reference
  never keeps the session alive (R156). It is also the identity check: vended IDs are
  reused (the counter resets to 1 when the last session is destroyed, which is what a
  restart does), so the ID alone could resolve to a newer session.

## Does
- resolve: the live parts, or nothing. The weak pointer's value must be non-nil, must
  still be the `LuaSession` registered under `vendedID` in `Server.luaSessions`, and must
  not be shut down; then the internal ID comes from `SessionManager.GetInternalID`.
  Every operation below goes through resolve.
- ID: returns vendedID (R154).
- Valid: resolve succeeds (R155).
- Lua / Browser / Exec: return a rig holding a copy of the reference; no work (R157,
  R165).
- Station: a struct holding all three rigs (R157).

### Rigs
Each rig is a struct holding the reference. Every rig has `Ref()` and `Station()`
(R158).
- ExecRig.Post(fn): resolve, then `WebSocketEndpoint.ExecuteInSessionPassive(internalID,
  ...)` with a closure that runs `fn` through `LuaSession.ExecuteInSession(vendedID, ...)`
  so the Lua context is set. Returns false without posting when resolve fails (R159,
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
- Server: issues references (`SessionRef`), holds `luaSessions` and the SessionManager
- WebSocketEndpoint: the session executor, change detection, connection state
- LuaSession: Lua context for posted work; shutdown state for validity
- SessionManager: vended-to-internal ID lookup

## Sequences
- seq-session-ref.md
