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
@alarm-hash: 37625d6f5bc2dedc
**Pulled:** 2026-11-18 — 2026-11-18 — rang: the historical defect restored, post sending from a spawned goroutine; "delivery order broken (R195): got [10 4 0 1 2 3 29 ...]"

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
@alarm-hash: 759c717aaa520245
**Pulled:** 2026-11-18 — 2026-11-18 — rang: accept launches even while a closure runs; "max concurrent closures = 8, want 1"

## Test: a closure may post fire-and-forget without deadlocking
**Purpose:** a post from inside a running closure queues behind it (R201)
**Input:** a closure that posts two more and returns; the actor is built without a cleanup Stop, so a wedged loop fails the test instead of hanging it
**Expected:** all three run in order within 2 seconds
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go
**Alarm:** 3
**Fire alarm:** run closures on the loop goroutine (call fn directly in launch, then next); the second inner post blocks on the one-slot channel the loop is not draining, and the test times out. One nested post is not enough: it fits in the empty slot, which is why the test posts two (found 2026-11-18; ark's version of this test posts one)
**Inject:** internal/server/svc.go:ChanSvc.launch
@alarm-hash: 25ced8d4daa8dedc
**Pulled:** 2026-11-18 — 2026-11-18 — rang only after the test was strengthened to post twice from inside: fn run on the loop in launch; "a closure posting fire-and-forget from inside itself deadlocked". With one nested post it stayed green

## Test: a burst larger than the channel is absorbed in order
**Purpose:** the unbounded queue absorbs bursts without reordering (R199)
**Input:** hold the first closure; post 199 more; release
**Expected:** all 200 run in post order within 5 seconds
**Refs:** crc-ChanSvc.md
**Code:** internal/server/svc_test.go

## Not tested, deliberately
**The panic policy (R200).** A panicking closure terminates the process, so testing it
in-process would take the test binary down. Its implementation is the absence of a recover;
this entry records that the absence is intentional.

**A synchronous post from inside a closure (R201).** The correct behaviour is a permanent
deadlock, so a test would hang rather than fail. Operation objects such as Tailor are the
audit surface.
