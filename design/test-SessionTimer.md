# Test Design: Session timers
**Source:** crc-LuaSession.md

Timer cancellation and its interaction with Shutdown. Cancel-twice is invisible
when it works and fatal when it does not: an interval's stop closes a channel,
so a second stop panics, and Shutdown cancels every registered handle, including
ones Lua already cleared.

## Test: a cleared interval survives Shutdown
**Purpose:** clearing a timer is a no-op once it is already cancelled (R102), including Shutdown's sweep of the registry (R109). Found 2026-10-21: ark's graceful stop hung because the mcp shell clears its `whenSys` interval and Shutdown then closed that channel a second time
**Input:** allocTimerHandle with an interval-style stop (`close(done)`); cancelTimer twice; Shutdown
**Expected:** no panic from the second cancel or from Shutdown
**Refs:** crc-LuaSession.md, seq-session-timer.md
**Code:** internal/lua/shutdown_race_test.go
**Fire alarm:** drop the `entry.cancelled` check in cancelTimer; the second cancel panics with "close of closed channel" and the test fails on its recover message
**Inject:** internal/lua/runtime.go:LuaSession.cancelTimer
**Pulled:** 2026-10-21 — rang: guard dropped; "cancelling a cleared interval again panicked: close of closed channel"
