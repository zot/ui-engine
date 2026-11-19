# Test Design: HostOp
**Source:** crc-HostOp.md

A host operation that works on the happy path can still be wrong in ways that pass by
default: a callback passed through to op as an argument, a delivery with no change
detection (the browser never hears), a failure that leaves the last success standing, a
deadline that never reaches work's context, and a backstop that never fires. Each gets
an alarm.

## Test: a call delivers its result to the fields and the callback
**Purpose:** the success path end to end; the callback is stripped from op's arguments (R205, R207, R208, R209, R211, R213, R215)
**Input:** define `greet` whose op records its arguments and whose work returns "hello " plus the first; from Lua, call `ui.greet(t, "msg", "bob", cb)` with `t.msgError` preset and cb recording its arguments
**Expected:** op saw exactly one argument; `t.msg` is "hello bob"; `t.msgError` is nil; cb got ("hello bob", nil); PendingTailors is empty
**Refs:** crc-HostOp.md, seq-host-op.md
**Code:** internal/server/hostop_test.go
**Alarm:** 1
**Fire alarm:** leave the trailing callback in the arguments passed to op; op sees two arguments
**Inject:** internal/server/hostop.go:LuaRig.callHostOp
@alarm-hash: 37ac484566bd811f
**Pulled:** 2026-11-19 — re-pulled after the rename to LuaRig: the trailing callback left in the arguments; "op saw 2 arguments, want 1 (the callback must not be passed)" *Earlier —* 2026-11-18 — rang: trailing callback left in the args; "op saw 2 arguments, want 1 (the callback must not be passed)"
## Test: delivery runs change detection
**Purpose:** the browser hears about the result (R212)
**Input:** a browser attached and a counting afterBatch; define `echo` whose work returns its argument; from Lua, call `ui.echo(t, "v", 1, cb)` and wait for cb
**Expected:** afterBatch ran after the call was made
**Refs:** crc-HostOp.md, seq-host-op.md
**Code:** internal/server/hostop_test.go
**Alarm:** 2
**Fire alarm:** deliver with the passive End instead of EndUpdate; afterBatch does not run after delivery
**Inject:** internal/server/hostop.go:hostOpCall.deliver
@alarm-hash: 00a5fc402983f8ab
**Pulled:** 2026-11-18 — rang: deliver used End instead of EndUpdate; "delivery ran no change detection (afterBatch 0 before, 0 after)"

## Test: a failing work clears the result and sets the error
**Purpose:** failure writes nil to the field and the message to the error field, and the callback gets it (R214, R215)
**Input:** define `fails` whose work returns an error; from Lua, preset `t.out` to "stale", call `ui.fails(t, "out", cb)`
**Expected:** `t.out` is nil; `t.outError` is the error message; cb got (nil, message)
**Refs:** crc-HostOp.md
**Code:** internal/server/hostop_test.go
**Alarm:** 3
**Fire alarm:** skip writing nil to the field on failure; `t.out` is still "stale"
**Inject:** internal/server/hostop.go:hostOpCall.deliver
@alarm-hash: 00a5fc402983f8ab
**Pulled:** 2026-11-18 — rang: field left unwritten on failure; `t.out = "stale", want "nil"`

## Test: an op error is delivered without running work
**Purpose:** (R210)
**Input:** define `refuses` whose op returns an error and a work that records it ran; call it from Lua
**Expected:** the error field holds op's message; work did not run
**Refs:** crc-HostOp.md, seq-host-op.md
**Code:** internal/server/hostop_test.go
**Alarm:** 4
**Fire alarm:** return without delivering when op errors; the error field is never written
**Inject:** internal/server/hostop.go:LuaRig.callHostOp
@alarm-hash: 37ac484566bd811f
**Pulled:** 2026-11-19 — re-pulled after the rename to LuaRig: return without delivering when op errors; "timed out waiting for refuses callback" *Earlier —* 2026-11-18 — rang: op error returned without delivering; "timed out waiting for refuses callback"
## Test: bad target or field raise at the call
**Purpose:** (R206)
**Input:** from Lua, `pcall(ui.greet, "not a table", "msg")` and `pcall(ui.greet, {}, 42)`
**Expected:** both pcalls return false
**Refs:** crc-HostOp.md
**Code:** internal/server/hostop_test.go

## Test: the call returns before the work finishes
**Purpose:** (R207)
**Input:** define `slow` whose work blocks until released; call it from a posted Lua chunk that signals when the call returns
**Expected:** the signal arrives while work is still blocked; after release the field is set
**Refs:** crc-HostOp.md, seq-host-op.md
**Code:** internal/server/hostop_test.go
**Alarm:** 5
**Fire alarm:** run work on the calling goroutine instead of its own; the call never returns
**Inject:** internal/server/hostop.go:LuaRig.callHostOp
@alarm-hash: 37ac484566bd811f
**Pulled:** 2026-11-19 — re-pulled after the rename to LuaRig: work run on the calling goroutine; "the host operation call did not return while its work was running" *Earlier —* 2026-11-18 — rang: work run on the calling goroutine; "the host operation call did not return while its work was running"
## Test: the deadline reaches work's context
**Purpose:** a work that honors its context fails with the context's error at the deadline (R211, R216)
**Input:** define `waits` with a 20ms deadline whose work returns `ctx.Err()` when its context is done; call it
**Expected:** the error field is "context deadline exceeded"
**Refs:** crc-HostOp.md
**Code:** internal/server/hostop_test.go
**Alarm:** 6
**Fire alarm:** give work the Tailor's context without the deadline; work is cancelled only by the backstop, whose late delivery is dropped, so the callback never runs
**Inject:** internal/server/hostop.go:hostOpCall.run
@alarm-hash: 5fef5bc4706b87df
**Pulled:** 2026-11-18 — rang: deadline narrowing disabled in run; "timed out waiting for waits callback"

## Test: a work that ignores its context is bounded and delivers nothing
**Purpose:** the Tailor closes as failed at twice the deadline and the late delivery is dropped (R216)
**Input:** define `stuck` with a 20ms deadline whose work ignores its context and blocks until released; call it; wait for PendingTailors to empty; release; barrier
**Expected:** PendingTailors empties within a second; the field and error field stay nil after release
**Refs:** crc-HostOp.md, crc-Tailor.md, seq-host-op.md
**Code:** internal/server/hostop_test.go
**Alarm:** 7
**Fire alarm:** create the Tailor with no deadline; PendingTailors never empties
**Inject:** internal/server/hostop.go:LuaRig.callHostOp
@alarm-hash: 37ac484566bd811f
**Pulled:** 2026-11-19 — re-pulled after the rename to LuaRig: Tailor created with no deadline; "the stuck operation's Tailor was never closed" *Earlier —* 2026-11-18 — rang: Tailor created with no deadline; "the stuck operation's Tailor was never closed"
