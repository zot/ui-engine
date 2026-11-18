# Closure Actor

**Language:** Go
**Environment:** ui-engine backend (`internal/server/svc.go`)

## Purpose

Each session has an executor, a closure actor (`ChanSvc`), that serializes everything
done to the session: browser messages, host calls, timers, and posted work. Exactly one
closure runs at a time.

Serialization is half of what makes it a closure actor. The other half is **order**,
which is not optional: an actor that loses it is broken. Closures posted by one
goroutine run in the order that goroutine posted them. Without it, the basic idiom of
posting a follow-up behind a message, instead of synchronizing against it, does not work,
because queue position stops meaning anything. Sewing, session timers (`setImmediate`
calls queue in order), and in-order handling of a browser's messages all depend on it.

This design is ported from ark, which found and fixed the same defect.

## Delivery

- **Order holds per posting goroutine,** for fire-and-forget and synchronous posts alike,
  since both enter by the same door. Posts from different goroutines interleave
  arbitrarily.
- **A post is a direct send on the calling goroutine.** The send is never handed to a
  freshly spawned goroutine: two such goroutines race, and the scheduler picks the winner.
- **The actor's own goroutine never runs a closure.** It keeps a queue and a flag saying a
  closure is in flight, both touched only on that goroutine. Each closure runs on a
  goroutine of its own and, when it finishes, posts back so the next one can start. Only
  one closure runs at any instant.
- **Bursts are absorbed by the queue, which is unbounded, not by the channel, which is
  one slot deep.** A sender blocks only while that slot is momentarily taken. One slot is
  a diagnostic choice: if the loop is ever wrongly given real work, the very next poster
  blocks at the site, instead of the stall surfacing later and somewhere else.
- **Panics are not recovered by the actor.** A panicking closure terminates the process: a
  panic inside the actor means the state it protects is in an unknown condition. A
  closure that can survive its own panic recovers internally.
- **A fire-and-forget post from inside a running closure is permitted**; it just queues
  behind the running closure. **A synchronous post from inside a running closure
  deadlocks**, and is forbidden.
- **The queue releases what it has run.** Popping clears the popped slot before
  advancing, so a run closure and everything it captured stop being reachable.
- **Stopping** posts a sentinel behind everything already posted. Closures still queued
  behind an in-flight one are abandoned. The channel is never closed, so a closure
  finishing after a stop does not panic when it posts back.

## API

```go
type ChanSvc struct{ /* unexported */ }

func NewChanSvc() *ChanSvc                                  // built, not running
func (s *ChanSvc) Start()                                   // launch the loop
func (s *ChanSvc) Stop()                                    // stop after what is already posted
func Svc(s *ChanSvc, fn func())                             // post, don't wait
func SvcSync[T any](s *ChanSvc, fn func() (T, error)) (T, error) // post and wait; never from inside a closure
```

The executor for a session is created and started on first use, and stopped when the
session's executor is cleaned up.
