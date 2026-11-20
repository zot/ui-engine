# Test Design: ExecAsync
**Source:** crc-ExecAsync.md

`ui.execAsync` runs a real program. As in test-PortableOS, the program is the test binary
itself, re-run as a helper child (the standard Go helper-process pattern), so the tests
need no Unix tool and run on Windows. The helper can echo its arguments, fail with
stderr and an exit status, or sleep.

## Test: every session has ui.execAsync before main.lua runs
**Purpose:** the operation is built in, defined before the app's own code (R250)
**Input:** a server whose `main.lua` records `type(ui.execAsync)` in a global as its first line
**Expected:** the recorded type is `function`
**Refs:** crc-ExecAsync.md, seq-session-create-backend.md
**Code:** internal/server/execasync_test.go
**Alarm:** 1
**Fire alarm:** define it after `CreateLuaSession` instead of before; main.lua records `nil`
**Inject:** internal/server/server.go:Server.CreateLuaBackendForSession
@alarm-hash: 0aeb3332c0b72fc3
**Pulled:** 2026-11-19 — re-pulled after Item 14 added SetExecutor to this function: rang `main.lua saw type(ui.execAsync) = nil, want function (R250)` *Earlier —* 2026-11-19 — rang: `main.lua saw type(ui.execAsync) = nil, want function (R250)`
## Test: stdout lands in the field and the callback
**Purpose:** a successful run delivers the program's stdout, verbatim argv, no shell (R266)
**Input:** `ui.execAsync(t, "out", {<helper>, "echo", "a b", "x>y"}, cb)`
**Expected:** `t.out` is `a b\nx>y\n`, `t.outError` is nil, and cb received the same stdout and no error
**Refs:** crc-ExecAsync.md
**Code:** internal/server/execasync_test.go
**Alarm:** 2
**Fire alarm:** return an empty string from the work instead of the stdout
**Inject:** internal/server/execasync.go:defineExecAsync
@alarm-hash: 240ec2871e23e7bc
**Pulled:** 2026-11-19 — re-pulled after the simplifier nested the error checks in the work: rang `t.out = "" (error nil), want "a b\nx>y\n"` *Earlier —* 2026-11-19 — rang: `t.out = "" (error nil), want "a b\nx>y\n"`
## Test: a failing program delivers its exit status and stderr
**Purpose:** failure reaches the error field with the reason (R266)
**Input:** the helper told to write `boom` to stderr and exit 3
**Expected:** `t.out` is nil and `t.outError` contains `3` and `boom`
**Refs:** crc-ExecAsync.md, crc-PortableOS.md
**Code:** internal/server/execasync_test.go

## Test: a program past its limit is killed and the call fails
**Purpose:** the time limit is enforced per call (R252)
**Input:** the helper told to sleep 30 seconds, with `argv.timeout = 0.2`
**Expected:** within 10 seconds, `t.outError` says the program timed out; `t.out` is nil
**Refs:** crc-ExecAsync.md
**Code:** internal/server/execasync_test.go
**Alarm:** 3
**Fire alarm:** ignore `argv.timeout` and use the 60-second default; the error does not arrive within 10 seconds
**Inject:** internal/lua/portable.go:ReadExec
@alarm-hash: ee40804fce3b283a
**Pulled:** 2026-11-19 — rang: `a 0.2-second limit did not end a 30-second program within 10 seconds (R252)`

## Test: the limit's default and its off switch
**Purpose:** absent `timeout` is 60 seconds, 0 is none, a non-table argv is an error (R252)
**Input:** `ReadExec` on `{"x"}`, `{"x", timeout = 0}`, `{}` and a string
**Expected:** 60s; 0; an error; an error
**Refs:** crc-PortableOS.md
**Code:** internal/lua/portable_test.go

## Covered elsewhere
Arguments passing verbatim and a failure's exit status and stderr (R266) are tested on
`ExecArgv`, which `ui.execAsync` runs, in test-PortableOS alarms 3 and 4. That `ui.exec` is
gone (R267) is test-PortableOS alarm 5.
