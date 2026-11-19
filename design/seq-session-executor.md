# Sequence: Session Executor

**Source Spec:** closure-actor.md
**Requirements:** R224, R225, R226, R227, R228

## Participants

- SessionManager: creates and destroys sessions
- Session: owns the executor
- ChanSvc: the session's executor
- Server: the destroy callback, which tears down the backend
- WebSocketEndpoint: posts browser messages and host work to the executor
- Poster: any goroutine posting work for a session

## Sequence: the executor is born with the session

```
1.1. SessionManager.CreateSession()
  1.1.1. NewSession(internalID): build exec, exec.Start()
  1.1.2. register the session and its vended ID
  1.1.3. onSessionCreated(vendedID, session)
    1.1.3.1. error -> forget the session, session.stopExecutor(); return the error
```

## Sequence: the executor ends with the session

```
2.1. SessionManager.DestroySession(id), on any goroutine, including the session's own executor
  2.1.1. forget the session and its IDs
  2.1.2. onSessionDestroyed -> Server.DestroyLuaBackendForSession: close Tailors, shut down
         Lua and the backend; the executor still runs
  2.1.3. session.stopExecutor() -> ChanSvc.Stop: the sentinel goes through send
2.2. the loop takes the sentinel: close done, return; the queue is abandoned
  2.2.1. the in-flight closure, if any, runs to completion; its send(next) is a no-op
  2.2.2. a SvcSync caller woken by done: launched -> wait for the result; else ErrStopped
```

## Sequence: work arrives for a session that is gone

```
3.1. Poster -> WebSocketEndpoint.ExecuteInSession* (or readPump with a message)
  3.1.1. executor(sessionID): SessionManager.GetSession; none -> nil
  3.1.2. nil -> ExecuteInSession returns "session not found"; Async, Passive and readPump drop
  3.1.3. found, but stopped meanwhile -> send returns false: Svc drops, SvcSync returns ErrStopped
```
