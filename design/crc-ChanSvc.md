# ChanSvc
**Requirements:** R195, R196, R197, R198, R199, R200, R201, R202, R203

The closure actor: serializes access to whatever its owner protects, and runs closures
posted by one goroutine in the order that goroutine posted them. Ported from ark's
`svc.go`. Its owner in ui-engine is WebSocketEndpoint, one actor per session.

Its distinguishing property is a split between keeping order and doing work. The loop
goroutine only manipulates the queue and the in-flight flag; every closure runs in a
goroutine of its own. That split is what lets a post be a direct send, and the direct send
is what makes order hold.

## Knows
- ch: the channel closures arrive on, one slot deep: a doorway, not the buffer (R199)
- queue: closures waiting for the in-flight one; unbounded; loop goroutine only (R197, R199)
- running: whether a closure is in flight; a plain bool, loop goroutine only (R197)

## Does
- NewChanSvc / Start: build, then launch the loop; the loop runs only bookkeeping
  closures (`accept`, `next`) and returns on the stop sentinel (R197).
- Svc(fn) / SvcSync(fn): post fn by a direct send on the calling goroutine (R195, R196).
  SvcSync also waits for fn; never call it from inside a running closure (R201).
- accept(fn) (loop): queue fn if a closure is in flight, else launch it (R197).
- launch(fn) (loop): mark running, run fn on its own goroutine, which posts `next` back
  when fn returns. No recover (R198, R200).
- next() (loop): pop the queue through popFront and launch, or clear running (R197, R202).
- Stop: post the sentinel behind everything already posted; never close the channel
  (R203).

## Collaborators
- WebSocketEndpoint: owns one ChanSvc per session; ChanSvc knows nothing about what it
  protects
- Tailor and other operation objects: the audit surface for the rule against synchronous
  posts from inside a closure (R201)

## Sequences
- seq-session-ref.md
- seq-tailor.md
