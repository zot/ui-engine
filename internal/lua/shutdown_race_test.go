// Regression tests for the LuaSession shutdown race: the executor goroutine
// must never touch the Lua VM after State.Close().
// CRC: crc-LuaSession.md | Seq: seq-session-timer.md | R115, R116, R117
package lua

import (
	"sync/atomic"
	"testing"
	"time"

	golua "github.com/yuin/gopher-lua"
	"github.com/zot/ui-engine/internal/config"
)

// TestShutdownJoinsInFlightExecutor verifies that Shutdown joins the executor
// before closing the VM: while a task is running inside the VM, Shutdown blocks;
// once the task finishes, Shutdown completes without a panic. R115.
func TestShutdownJoinsInFlightExecutor(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})

	// Register a Go function that blocks while "inside the VM" — standing in
	// for the app-loader running app Lua during a deferred task.
	rt.execute(func() (interface{}, error) {
		rt.State.SetGlobal("blockInVM", rt.State.NewFunction(func(L *golua.LState) int {
			close(entered)
			<-release
			return 0
		}))
		return nil, nil
	})

	// Dispatch a task that enters the VM and blocks there.
	go rt.execute(func() (interface{}, error) {
		return nil, rt.State.CallByParam(golua.P{
			Fn:      rt.State.GetGlobal("blockInVM"),
			NRet:    0,
			Protect: true,
		})
	})

	<-entered // executor is now inside the VM

	shutdownReturned := make(chan struct{})
	go func() {
		rt.Shutdown()
		close(shutdownReturned)
	}()

	// The join must block while the executor is mid-VM; Shutdown must not close
	// the VM underneath the running task.
	select {
	case <-shutdownReturned:
		t.Fatal("Shutdown returned while executor was still in the VM (would race State.Close)")
	case <-time.After(100 * time.Millisecond):
	}

	close(release) // let the in-VM task finish

	select {
	case <-shutdownReturned:
		// join completed, VM closed cleanly, no panic
	case <-time.After(shutdownJoinTimeout + time.Second):
		t.Fatal("Shutdown did not complete after the in-flight task finished")
	}
}

// TestExecutorSkipsQueuedWorkAfterShutdown verifies that work queued on the
// executor channel during shutdown is not run; the executor returns the
// shutdown error instead. R116.
func TestExecutorSkipsQueuedWorkAfterShutdown(t *testing.T) {
	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}

	// Occupy the executor with a blocking task.
	entered := make(chan struct{})
	release := make(chan struct{})
	firstResult := make(chan WorkResult, 1)
	rt.executorChan <- WorkItem{
		fn: func() (interface{}, error) {
			close(entered)
			<-release
			return nil, nil
		},
		result: firstResult,
	}
	<-entered // executor busy in the first task

	// Queue a second task that records whether it ran.
	var ranSecond atomic.Bool
	secondResult := make(chan WorkResult, 1)
	rt.executorChan <- WorkItem{
		fn: func() (interface{}, error) {
			ranSecond.Store(true)
			return nil, nil
		},
		result: secondResult,
	}

	// Signal shutdown, then release the first task.
	close(rt.done)
	close(release)
	<-firstResult        // first task finished
	<-rt.executorStopped // executor has observed done and returned

	if ranSecond.Load() {
		t.Fatal("queued work ran after shutdown was signaled")
	}
	// If the executor dequeued the second item, it must have returned the
	// shutdown error rather than running it.
	select {
	case res := <-secondResult:
		if res.Err != errSessionShutdown {
			t.Fatalf("expected errSessionShutdown for skipped work, got %v", res.Err)
		}
	default:
		// Executor took the done-return branch; the item was left unprocessed.
		// Either way it did not run — the guarantee holds.
	}

	// Executor has returned; close the VM directly. (We drove `done` manually,
	// so we must not call Shutdown — that would double-close the channel.)
	rt.State.Close()
}

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

	// Shutdown signaled before the deferred task gets its turn.
	close(rt.done)
	<-rt.executorStopped // executor returned; safe to close the VM at the end

	if _, err := wrapped(); err != nil {
		t.Fatalf("deferred wrapper returned error: %v", err)
	}
	if ran.Load() {
		t.Fatal("deferred task touched the VM after shutdown was signaled")
	}

	rt.State.Close()
}
