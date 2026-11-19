// Regression tests for LuaSession shutdown: nothing may touch the Lua VM after
// State.Close(). (The join and queue-skip tests went with the separate Lua executor
// goroutine, Item 14; test-OneExecutor.md covers shutdown on the session executor.)
// CRC: crc-LuaSession.md | Seq: seq-session-timer.md | R102, R109, R117
package lua

import (
	"sync/atomic"
	"testing"

	golua "github.com/yuin/gopher-lua"
	"github.com/zot/ui-engine/internal/config"
)

// TestDeferredWrapperSkipsAfterShutdown verifies that a deferred-task wrapper
// does not touch the VM once shutdown has been signaled. R117.
func TestDeferredWrapperSkipsAfterShutdown(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	rt.SetVariableStore(newMockStore())

	// Capture the wrapped deferred task instead of dispatching it.
	var wrapped func() (interface{}, error)
	rt.onDefer = func(fn func() (interface{}, error)) { wrapped = fn }

	// Hold the VM close back, as the Server does until its executor drains, so the
	// wrapper below runs with the session shut down but the VM still open.
	var closeVM func()
	rt.SetExecutor(nil, func(c func()) { closeVM = c })

	var ran atomic.Bool
	var luaFn *golua.LFunction
	rt.execute(func() (interface{}, error) {
		luaFn = rt.State.NewFunction(func(L *golua.LState) int {
			ran.Store(true)
			return 0
		})
		return nil, nil
	})

	rt.scheduleDeferred(1, luaFn)
	if wrapped == nil {
		t.Fatal("scheduleDeferred did not call onDefer")
	}

	// Shutdown before the deferred task gets its turn.
	rt.Shutdown()

	if _, err := wrapped(); err != nil {
		t.Fatalf("deferred wrapper returned error: %v", err)
	}
	if ran.Load() {
		t.Fatal("deferred task touched the VM after shutdown was signaled")
	}

	if closeVM == nil {
		t.Fatal("Shutdown never handed the VM close to closeWhenDrained")
	}
	closeVM()
}

// TestClearedIntervalSurvivesShutdown: an interval Lua already cleared stays in
// the registry, and Shutdown cancels every registered handle. Cancelling must be
// a no-op the second time, since an interval's stop closes a channel. Before the
// fix this panicked with "close of closed channel" inside Shutdown, and ark's
// graceful stop never finished. R102, R109
func TestClearedIntervalSurvivesShutdown(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("cancelling a cleared interval again panicked: %v", p)
		}
	}()
	done := make(chan struct{})
	handle := rt.allocTimerHandle(func() { close(done) })
	rt.cancelTimer(handle) // clearInterval
	rt.cancelTimer(handle) // a second clear is a no-op too
	rt.Shutdown()          // and so is Shutdown's sweep of the registry
}
