# Sequence: Session references and dangerous execution

**Source Spec:** session-ref.md
**Requirements:** R153, R155, R159, R160, R161, R168, R169, R170, R171

## Participants

- Caller: host code holding a SessionRef, or calling ExecuteInSession
- Server: issues references, PrimitiveExecuteInSession, luaSessions registry
- SessionRef: resolve, rigs
- WSEndpoint: WebSocketEndpoint, the per-session ChanSvc executor and afterBatch
- LuaSession: Lua context and executor

## Sequence: posting through a reference

```
1.1. Caller -> Server.SessionRef(vendedID)
  1.1.1. Server looks up luaSessions[vendedID]; none -> return (zero ref, false)
  1.1.2. Server returns SessionRef{srv, vendedID, weak.Make(luaSession)}, true
1.2. Caller -> ref.Exec().Post(fn)   (PostUpdate differs only at 1.2.3)
  1.2.1. SessionRef.resolve: weak value non-nil, still luaSessions[vendedID], not shut down
  1.2.2. resolve fails -> return false; nothing is queued
  1.2.3. Post -> WSEndpoint.ExecuteInSessionPassive(internalID, closure)
         PostUpdate -> WSEndpoint.ExecuteInSessionAsync(internalID, closure)
  1.2.4. return true without waiting
1.3. WSEndpoint executor runs closure
  1.3.1. closure -> LuaSession.ExecuteInSession(vendedID, fn); Lua context set, fn runs
  1.3.2. passive: done. active: afterBatch when a browser is connected
```

## Sequence: dangerous execution

```
2.1. Caller -> Server.ExecuteInSession(vendedID, fn)
  2.1.1. ExecuteInSession -> PrimitiveExecuteInSession(vendedID, fn, true)
2.2. PrimitiveExecuteInSession(vendedID, fn, dangerous)
  2.2.1. dangerous: find the site that called ExecuteInSession (runtime.Caller)
  2.2.2. AllowDangerousExecution set: first time for this site -> log a warning; continue
  2.2.3. AllowDangerousExecution not set: panic naming the site; nothing is queued
  2.2.4. queue through WSEndpoint.ExecuteInSession -> LuaSession.ExecuteInSession, wait
  2.2.5. return fn's result and error
```
