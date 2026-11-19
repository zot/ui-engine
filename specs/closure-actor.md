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
- **Stopping is final, and never blocks anyone.** A closure in flight when the actor
  stops runs to completion; closures still queued behind it are abandoned. Once stopped,
  the actor's goroutine has ended, and every post is dropped at once instead of waiting
  on a channel nobody drains: a fire-and-forget post returns, and a synchronous post
  returns `ErrStopped`. A synchronous caller whose closure was abandoned in the queue
  also returns `ErrStopped` rather than waiting forever. A closure finishing after the
  stop does not block or panic when it reports back. Stopping twice is harmless, and
  stopping from inside a running closure is allowed, since that closure runs off the
  actor's goroutine.

## API

```go
type ChanSvc struct{ /* unexported */ }

var ErrStopped error                                        // a synchronous post to a stopped actor

func NewChanSvc() *ChanSvc                                  // built, not running
func (s *ChanSvc) Start()                                   // launch the loop
func (s *ChanSvc) Stop()                                    // final; abandon what has not started
func Svc(s *ChanSvc, fn func())                             // post, don't wait; dropped once stopped
func SvcSync[T any](s *ChanSvc, fn func() (T, error)) (T, error) // post and wait; never from inside a closure
```

## The session's executor

A session's executor belongs to the session: it is created and started when the session
is created, and stopped when the session is destroyed, after the session's backend has
been torn down, so that teardown can still use it. Destroying a session from inside its
own executor is allowed and completes. When a session is gone, nothing of its executor
remains running: the goroutine count after a destroyed session is what it was before the
session was created.

Work for a session that does not exist creates no executor. A synchronous execution
returns an error; fire-and-forget work is dropped. This includes work that arrives after
the session is destroyed, such as a late browser message, a timer, or a host operation
delivering its result. None of these may block the goroutine that posted it.
