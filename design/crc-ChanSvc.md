# ChanSvc
**Requirements:** R195, R196, R197, R198, R199, R200, R201, R202, R218, R219, R220, R221, R222, R223

The closure actor: serializes access to whatever its owner protects, and runs closures
posted by one goroutine in the order that goroutine posted them. Ported from ark's
`svc.go`. Its owner in ui-engine is Session, one actor per session.

Its distinguishing property is a split between keeping order and doing work. The loop
goroutine only manipulates the queue and the in-flight flag; every closure runs in a
goroutine of its own. That split is what lets a post be a direct send, and the direct send
is what makes order hold.

## Knows
- ch: the channel bookkeeping closures arrive on, one slot deep: a doorway, not the
  buffer (R199)
- done: closed by the loop goroutine, and only by it, when it takes the stop sentinel;
  closed means stopped (R219)
- queue: jobs waiting for the in-flight one; unbounded; loop goroutine only (R197, R199)
- running: whether a job is in flight; a plain bool, loop goroutine only (R197)
- job: one posted closure, plus `launched`, written only on the loop goroutine when the
  job starts. Every launch happens before `done` closes, on the same goroutine, so a
  reader woken by `done` reads `launched` without a race (R218, R221)
- ErrStopped: what a synchronous post to a stopped actor returns (R220, R221)

## Does
- NewChanSvc / Start: build, then launch the loop. The loop receives from ch and runs only
  bookkeeping closures (`accept`, `next`); on the stop sentinel it closes done and returns,
  abandoning the queue (R197, R218, R219).
- send(f): the one way anything enters ch: select between sending f and done; reports
  whether f was sent. Once stopped it returns false at once, never blocking (R219, R222).
- Svc(fn): post a job through send; dropped when stopped (R195, R196, R219).
- SvcSync(fn): post a job through send; not sent -> ErrStopped (R220). Then wait for the
  result or done. Woken by done: the job launched -> it is in flight and runs to
  completion, so keep waiting for its result (R218); not launched -> it was abandoned,
  return ErrStopped (R221). Never call it from inside a running closure (R201).
- accept(job) (loop): queue the job if one is in flight, else launch it (R197).
- launch(job) (loop): set launched and running, run the closure on its own goroutine,
  which reports back with `send(next)` when it returns; after a stop that send is a
  no-op (R198, R200, R222).
- next() (loop): pop the queue through popFront and launch, or clear running (R197, R202).
- Stop: send the sentinel through send. A later Stop finds done closed and returns, and a
  second sentinel sent before the loop took the first lies unread in the slot, so Stop is
  idempotent with no flag of its own. Callable from inside a running closure, which runs
  off the loop goroutine (R223).

## Collaborators
- Session: owns one ChanSvc, created with the session and stopped when it is destroyed;
  ChanSvc knows nothing about what it protects
- Tailor and other operation objects: the audit surface for the rule against synchronous
  posts from inside a closure (R201)

## Sequences
- seq-session-ref.md
- seq-tailor.md
- seq-session-executor.md
