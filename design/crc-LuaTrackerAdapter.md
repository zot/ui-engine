# LuaTrackerAdapter
**Requirements:** R239, R240, R241, R242, R243

**Source Spec:** session-ref.md ("Where a session's parts live")

The server's implementation of `lua.VariableStore` (`luaTrackerAdapter` in
`internal/server/server.go`): it lets a Lua session create, read, update and destroy
variables in its own session's tracker. It holds no per-session state of its own; every
operation names its session and finds that session's backend through the registry.

## Knows
- config, viewdefManager: logging and viewdef delivery
- sessions: the SessionManager, the registry it resolves sessions through (R239)

## Does
- backend(vendedID): `sessions.GetByVendedID`, then the Session's backend as a
  `*backend.LuaBackend`; nil when there is no such session or it has no Lua backend. The
  one way the adapter reaches a session's backend; it keeps no copy (R239).
- CreateSession(vendedID, resolver): set the resolver on that session's tracker. The
  negative-ID counter is the backend's, so there is nothing else to set up (R240).
- DestroySession(vendedID): nothing to clean up; the session's state goes with the session.
- GetTracker(vendedID), DetectChanges(vendedID), GetChanges(vendedID): through backend.
- CreateVariable(vendedID, parentID, object, properties): root (parent 0) takes the
  tracker's ID 1; any other server variable takes `backend.NextServerVarID()`, -1 and
  down, counted per session (R240, R45).
- Get(vendedID, id), GetProperty(vendedID, id, name), Update(vendedID, id, value,
  properties), Destroy(vendedID, id): find the backend for the named session, then the
  variable in its tracker. There is no variable-to-session index: variable IDs repeat
  across sessions, so the caller names the session (R241). Destroy is what
  `session:destroyVariable` reaches (R243).
- updateVariable1Viewdefs(vendedID, viewdefs): through backend.

## Collaborators
- SessionManager: the session registry
- Session: holds the backend
- LuaBackend: the session's tracker and its server-variable counter
- LuaSession: the only caller; it names its own session in every operation (R242)

## Sequences
- seq-session-create-backend.md
- seq-lua-session-init.md
