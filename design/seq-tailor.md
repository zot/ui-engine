# Sequence: Tailor

**Source Spec:** tailor.md
**Requirements:** R174, R176, R177, R178, R179, R180, R181, R183, R184, R185, R189, R191, R193

## Participants

- Worker: host code, on any goroutine
- Tailor: one sewing operation
- ExecView: posts to the session executor
- Executor: the session's executor running a stitch
- Session: holds the pending list
- Server: destroys sessions

## Sequence: a sewing

```
1.1. Worker -> ExecView.Tailor(name, deadline)
  1.1.1. Tailor created with ctx; deadline timer started unless deadline is 0
  1.1.2. Session.addTailor(t)
1.2. Worker -> t.Sew(fn, updateOnErr)   (SewUpdate / End / EndUpdate differ only in flags)
  1.2.1. under mu: ended -> panic; closed -> log once, return
  1.2.2. record lastStitch and lastSite; terminal -> ended = true
  1.2.3. ref invalid -> close as failed; return
  1.2.4. passive -> ExecView.Post(runStitch); active -> ExecView.PostUpdate(runStitch)
1.3. Executor runs runStitch
  1.3.1. closed by failure, deadline or session end -> skip
  1.3.2. set state and inStitch; run fn with recover; clear them
  1.3.3. error -> log with name; close as failed; passive with updateOnErr -> BrowserView.Update
  1.3.4. terminal -> close
  1.3.5. active -> change detection after the stitch (ExecView.PostUpdate)
```

## Sequence: closing without the terminal stitch

```
2.1. deadline timer fires
  2.1.1. t.close("deadline exceeded", failed); log name and last site
2.2. Server.DestroyLuaBackendForSession(vendedID, sess)
  2.2.1. for each Tailor on sess's pending list: close("session destroyed", failed)
  2.2.2. each close cancels ctx, stops the timer, leaves the pending list, logs name, age, last site
```
