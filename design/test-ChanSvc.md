# Test Design: ChanSvc
**Source:** crc-ChanSvc.md

Ported from ark's test-ChanSvc.md. The order guarantee was assumed in ui-engine, was in
fact absent, and nothing failed until Tailor's tests ran stitches out of order. These are
written to fail if the delivery path loses it again.

## Test: posts from one goroutine run in post order
**Purpose:** the core guarantee (R195, R196). A spawned sender per post fails this non-deterministically, which is exactly the regression to catch
**Input:** one goroutine posts 500 closures that each record their index; wait for the last
**Expected:** the recorded sequence is 0..499
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go
**Alarm:** 1
**Fire alarm:** restore the historical defect, sending from a spawned goroutine in post; the recorded order breaks
**Inject:** internal/server/svc.go:ChanSvc.post
@alarm-hash: 816a38546f8b8fcf
**Pulled:** 2026-11-19 — re-pulled after the job rewrite: post sending from a spawned goroutine; "delivery order broken (R195): got [3 1 2 4 0 102 6 ...]" *Earlier —* 2026-11-18 — 2026-11-18 — rang: the historical defect restored, post sending from a spawned goroutine; "delivery order broken (R195): got [10 4 0 1 2 3 29 ...]"
## Test: ordering spans fire-and-forget and synchronous posts
**Purpose:** both post kinds enter by the same door, so their interleaving keeps order (R195)
**Input:** from one goroutine, alternate Svc and SvcSync posts that record their index
**Expected:** the recorded sequence matches the post sequence
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go

## Test: exactly one closure runs at a time
**Purpose:** serialization is held by the in-flight flag now that work runs off the loop (R197, R198)
**Input:** 8 concurrent posters, 25 SvcSync posts each; each closure tracks concurrent entries and sleeps 1ms
**Expected:** maximum observed concurrency is 1
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go
**Alarm:** 2
**Fire alarm:** in accept, launch even when a closure is running; concurrency exceeds 1
**Inject:** internal/server/svc.go:ChanSvc.accept
@alarm-hash: bc44e7901ba7739a
**Pulled:** 2026-11-19 — re-pulled after Item 14 added the stopping state to accept: rang `max concurrent closures = 8, want 1 (R197, R198)` *Earlier —* 2026-11-19 — re-pulled after the job rewrite: accept launches even while a job runs; "max concurrent closures = 8, want 1" *Earlier —* 2026-11-18 — 2026-11-18 — rang: accept launches even while a closure runs; "max concurrent closures = 8, want 1"
## Test: a closure may post fire-and-forget without deadlocking
**Purpose:** a post from inside a running closure queues behind it (R201)
**Input:** a closure that posts two more and returns; the actor is built without a cleanup Stop, so a wedged loop fails the test instead of hanging it
**Expected:** all three run in order within 2 seconds
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go
**Alarm:** 3
**Fire alarm:** run closures on the loop goroutine (call fn directly in launch, then next); the second inner post blocks on the one-slot channel the loop is not draining, and the test times out. One nested post is not enough: it fits in the empty slot, which is why the test posts two (found 2026-11-18; ark's version of this test posts one)
**Inject:** internal/server/svc.go:ChanSvc.launch
@alarm-hash: 2a56f969e33d7f96
**Pulled:** 2026-11-19 — re-pulled after Item 14 made the report-back a plain send: rang `a closure posting fire-and-forget from inside itself deadlocked (R201)` *Earlier —* 2026-11-19 — re-pulled after the job rewrite: the job run on the loop goroutine in launch, then next; "a closure posting fire-and-forget from inside itself deadlocked (R201)" *Earlier —* 2026-11-18 — 2026-11-18 — rang only after the test was strengthened to post twice from inside: fn run on the loop in launch; "a closure posting fire-and-forget from inside itself deadlocked". With one nested post it stayed green
## Test: a burst larger than the channel is absorbed in order
**Purpose:** the unbounded queue absorbs bursts without reordering (R199)
**Input:** hold the first closure; post 199 more; release
**Expected:** all 200 run in post order within 5 seconds
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go

## Test: posts after a stop return at once
**Purpose:** a stopped actor never blocks a poster (R219, R220). Before 2026-11-19 the loop simply returned, so the first late post filled the one-slot channel and every later one blocked forever
**Input:** start and stop an actor; then, under a 2-second deadline, post 5 fire-and-forget closures and one SvcSync
**Expected:** every post returns before the deadline; no closure runs; SvcSync returns `ErrStopped`
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go
**Alarm:** 4
**Fire alarm:** make send a plain `s.ch <- f` without the select on done; the second late post blocks and the deadline fails the test
**Inject:** internal/server/svc.go:ChanSvc.send
@alarm-hash: 72fb77050aabf0e3
**Pulled:** 2026-11-19 — rang: send made a plain `s.ch <- fn` with no select on done; "posting to a stopped actor did not return within 2s"

## Test: an abandoned synchronous caller gets ErrStopped
**Purpose:** a SvcSync whose job was queued behind the in-flight one when the actor stopped does not wait forever (R218, R221)
**Input:** hold a first closure in flight; post a SvcSync from another goroutine so it queues; Stop; release the first closure
**Expected:** the SvcSync caller returns `ErrStopped` within 2 seconds; its closure never ran; the held closure ran to completion
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go
**Alarm:** 5
**Fire alarm:** in SvcSync, wait only for the result, not for done; the abandoned caller hangs and the deadline fails the test
**Inject:** internal/server/svc.go:SvcSync
@alarm-hash: 86fd10c35ef2d4ae
**Pulled:** 2026-11-19 — rang: SvcSync waits only for its result, not for done; "a SvcSync abandoned by Stop waited forever (R221)"

## Test: a synchronous caller whose closure stops the actor gets its result
**Purpose:** a job in flight runs to completion, so the caller of a closure that calls Stop gets that closure's result, not `ErrStopped` (R218, R223). This is the shape of a session destroying itself from inside its executor
**Input:** SvcSync a closure that calls Stop, sleeps 10ms so done closes first, and returns 42
**Expected:** SvcSync returns 42 and no error
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go
**Alarm:** 6
**Fire alarm:** in SvcSync, return `ErrStopped` whenever done wakes it, ignoring launched; the caller gets `ErrStopped`
**Inject:** internal/server/svc.go:SvcSync
@alarm-hash: 86fd10c35ef2d4ae
**Pulled:** 2026-11-19 — rang: SvcSync returns ErrStopped whenever done wakes it, ignoring launched; "got (0, executor stopped), want (42, nil)"

## Test: stopping twice is harmless
**Purpose:** Stop is idempotent (R223)
**Input:** start an actor; call Stop three times, under a 2-second deadline
**Expected:** every call returns; a later post returns at once
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go
**Alarm:** 7
**Fire alarm:** make Stop a plain `s.ch <- nil` that bypasses send; after the loop has ended the second sentinel fills the empty slot and the third blocks, failing the deadline. Three calls, not two, because two pass with the defect present
**Inject:** internal/server/svc.go:ChanSvc.Stop
@alarm-hash: 1016843207ade23f
**Pulled:** 2026-11-19 — rang: Stop a plain `s.ch <- nil` bypassing send; "three Stops and a post did not return within 2s"

## Test: stopping ends the loop goroutine
**Purpose:** the goroutine count returns after a stop (R219)
**Input:** record `runtime.NumGoroutine`; start 50 actors, run one closure on each, stop them all
**Expected:** within 2 seconds the count is back to the recorded value
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go

## Test: drained waits for the closure in flight at the stop
**Purpose:** drained means stopped and the last closure returned, not just stopped (R263). The loop closes done the moment it takes the stop sentinel, while an in-flight closure runs on, so done alone would let a caller close what the actor protects under that closure
**Input:** a closure that stops the actor from inside, then sleeps 100 ms and records that it returned; then an idle actor that is stopped
**Expected:** Drained() is not closed before the closure records it returned, and is closed within 2 seconds after; for the idle actor Drained() closes within 2 seconds of the stop
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go
**Alarm:** 8
**Fire alarm:** close drained at the stop sentinel regardless of a closure in flight; Drained() closes before the closure has returned
**Inject:** internal/server/svc.go:ChanSvc.Start
@alarm-hash: e7390673259dca8a
**Pulled:** 2026-11-19 — rang: `Drained closed before the closure in flight at the stop returned`

## Not tested, deliberately
**The panic policy (R200).** A panicking closure terminates the process, so testing it
in-process would take the test binary down. Its implementation is the absence of a recover;
this entry records that the absence is intentional.

**A synchronous post from inside a closure (R201).** The correct behaviour is a permanent
deadlock, so a test would hang rather than fail. Operation objects such as Tailor are the
audit surface.
