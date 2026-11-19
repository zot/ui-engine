# Test Design: Tailor
**Source:** crc-Tailor.md

Most Tailor properties fail quietly: a chain that keeps going after an error still
produces output, a passive stitch that runs change detection only costs time, a pending
entry that is never removed only grows a list, and a context that is never cancelled only
leaks a child process. Each gets an alarm.

## Test: a sewing runs its stitches in order and closes
**Purpose:** stitches post without waiting, run in order with the Lua context, and End closes the Tailor and removes it from the pending list (R176, R177, R183, R191)
**Input:** a Tailor; Sew a stitch that appends "a" and checks `State()` is non-nil; End a stitch that appends "b"; wait for End
**Expected:** order is a, b; the Tailor is not valid; PendingTailors is empty; Context is done
**Refs:** crc-Tailor.md, seq-tailor.md
**Code:** internal/server/tailor_test.go
**Alarm:** 1
**Fire alarm:** skip leaving the pending list in close; PendingTailors still lists the Tailor
**Inject:** internal/server/tailor.go:Tailor.close
@alarm-hash: 3c1d4d6b984c4a87
**Pulled:** 2026-11-18 — 2026-11-18 — rang: close skipped removeTailor; "PendingTailors lists 1 after End, want 0"

## Test: an error ends the sewing
**Purpose:** a stitch that returns an error closes the Tailor as failed, and a stitch already queued behind it does not run (R181)
**Input:** Sew a stitch returning an error, then Sew a stitch that records it ran, then post a barrier through the ExecRig
**Expected:** the second stitch did not run; the Tailor is not valid; the log names the Tailor
**Refs:** crc-Tailor.md, seq-tailor.md
**Code:** internal/server/tailor_test.go
**Alarm:** 2
**Fire alarm:** remove the closed check at the start of runStitch; the second stitch runs
**Inject:** internal/server/tailor.go:Tailor.runStitch
@alarm-hash: 1160f79828fe137f
**Pulled:** 2026-11-18 — 2026-11-18 — rang: runStitch's closed check removed; "a stitch queued behind a failed stitch ran"

## Test: a panic counts as an error
**Purpose:** a panicking stitch is recovered and closes the Tailor as failed (R180)
**Input:** Sew a stitch that panics; barrier
**Expected:** the test process survives; the Tailor is not valid
**Refs:** crc-Tailor.md
**Code:** internal/server/tailor_test.go

## Test: updateOnErr runs change detection only on error
**Purpose:** a passive Sew runs change detection after an error only with updateOnErr (R182)
**Input:** a browser attached and a counting afterBatch; one Tailor whose Sew errors with updateOnErr false, another whose Sew errors with updateOnErr true; barrier
**Expected:** afterBatch count 0 for the first, 1 for the second
**Refs:** crc-Tailor.md, crc-SessionRef.md
**Code:** internal/server/tailor_test.go
**Alarm:** 3
**Fire alarm:** call BrowserRig.Update on every error regardless of updateOnErr; the first Tailor's count is 1
**Inject:** internal/server/tailor.go:Tailor.runStitch
@alarm-hash: 1160f79828fe137f
**Pulled:** 2026-11-18 — 2026-11-18 — rang: Update called on every passive error; "afterBatch ran 1 times for an error without updateOnErr, want 0"

## Test: posting after End panics
**Purpose:** misuse is loud (R184)
**Input:** End a stitch, then Sew another, under recover
**Expected:** the Sew panics
**Refs:** crc-Tailor.md
**Code:** internal/server/tailor_test.go
**Alarm:** 4
**Fire alarm:** drop the ended check in post; no panic
**Inject:** internal/server/tailor.go:Tailor.post
@alarm-hash: 3184f6f5b3eb49b1
**Pulled:** 2026-11-18 — 2026-11-18 — rang: post's ended check removed; "Sew after End did not panic"

## Test: posting after a failure logs once and drops
**Purpose:** lateness is not a panic, and is logged once (R185)
**Input:** a Tailor failed by an erroring stitch; Sew twice more with stitches that record they ran; barrier
**Expected:** no panic; neither stitch ran; exactly one late-post log line naming the Tailor
**Refs:** crc-Tailor.md
**Code:** internal/server/tailor_test.go
**Alarm:** 5
**Fire alarm:** stop setting warnedLate; two late-post lines appear
**Inject:** internal/server/tailor.go:Tailor.post
@alarm-hash: 3184f6f5b3eb49b1
**Pulled:** 2026-11-18 — 2026-11-18 — rang: warnedLate never set; "late-post logged 2 times, want 1"

## Test: State is only available inside a stitch
**Purpose:** State panics outside a running stitch of its Tailor (R187)
**Input:** call State() from the test goroutine, under recover
**Expected:** a panic
**Refs:** crc-Tailor.md
**Code:** internal/server/tailor_test.go
**Alarm:** 6
**Fire alarm:** return the LState without checking inStitch; no panic
**Inject:** internal/server/tailor.go:Tailor.State
@alarm-hash: eb32c4dbfa58f4aa
**Pulled:** 2026-11-18 — 2026-11-18 — rang: State's inStitch check removed; "State outside a stitch did not panic"

## Test: the deadline fails the Tailor and cancels its context
**Purpose:** a non-zero deadline closes the Tailor as failed and cancels its context; a deadline of 0 never expires (R175, R189, R190)
**Input:** one Tailor with a 20ms deadline and one long-lived; wait for the first's Context
**Expected:** the first's Context is done within a second and it is not valid; the long-lived one is still valid, listed with LongLived true
**Refs:** crc-Tailor.md, seq-tailor.md
**Code:** internal/server/tailor_test.go
**Alarm:** 7
**Fire alarm:** do not cancel the context in close; the wait times out
**Inject:** internal/server/tailor.go:Tailor.close
@alarm-hash: 3c1d4d6b984c4a87
**Pulled:** 2026-11-18 — 2026-11-18 — rang: close stopped cancelling the context; "deadline passed without cancelling the context"

## Test: destroying the session closes its Tailors
**Purpose:** session destroy closes every open Tailor as failed and cancels its context (R193)
**Input:** a Tailor and a long-lived Tailor; destroy the session
**Expected:** both Contexts are done; neither is valid
**Refs:** crc-Tailor.md, crc-Server.md, seq-tailor.md
**Code:** internal/server/tailor_test.go
**Alarm:** 8
**Fire alarm:** remove the pending-list sweep from DestroyLuaBackendForSession; the long-lived Tailor's Context is never done
**Inject:** internal/server/server.go:Server.DestroyLuaBackendForSession
@alarm-hash: f3a167a944e0ee41
**Pulled:** 2026-11-19 — re-pulled after DestroyLuaBackendForSession dropped the map: pending-list sweep removed; `Tailor "timed" context not cancelled by session destroy` *Earlier —* 2026-11-18 — 2026-11-18 — rang: destroy sweep removed; the timed Tailor's context was not cancelled by session destroy
## Test: posting on an ended session closes the Tailor
**Purpose:** a stitch posted after the session is gone closes the Tailor as failed without posting (R179, R186)
**Input:** a Tailor whose session's Lua session is shut down without the destroy sweep; Sew a stitch that records it ran
**Expected:** the stitch does not run; the Tailor is not valid
**Refs:** crc-Tailor.md
**Code:** internal/server/tailor_test.go

## Test: the pending list reports the last calling site
**Purpose:** PendingTailors lists open Tailors with name and the last post's file and line (R178, R192)
**Input:** a long-lived Tailor; Sew a no-op stitch from this file; barrier
**Expected:** one entry with the Tailor's name and a LastSite naming tailor_test.go
**Refs:** crc-Tailor.md, crc-SessionRef.md
**Code:** internal/server/tailor_test.go
**Alarm:** 9
**Fire alarm:** record the site one frame too shallow (inside tailor.go); LastSite names tailor.go
**Inject:** internal/server/tailor.go:Tailor.post
@alarm-hash: 3184f6f5b3eb49b1
**Pulled:** 2026-11-18 — 2026-11-18 — rang: callerSite(2) in post; "LastSite .../tailor.go:105 does not name the calling test file"
