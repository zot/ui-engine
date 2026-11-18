// CRC: crc-HostOp.md | Test: test-HostOp.md
package server

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gopher "github.com/yuin/gopher-lua"
)

// hostOpTest is a session with a Lua global signal(name) that closes the named
// channel, so Lua callbacks can tell the test they ran.
type hostOpTest struct {
	t   *testing.T
	s   *Server
	ref SessionRef
	id  string

	mu      sync.Mutex
	signals map[string]chan struct{}
}

func newHostOpTest(t *testing.T) *hostOpTest {
	t.Helper()
	s, ref, internalID := newTailorTest(t)
	h := &hostOpTest{t: t, s: s, ref: ref, id: internalID, signals: map[string]chan struct{}{}}
	h.onExec(func(L *gopher.LState) {
		L.SetGlobal("signal", L.NewFunction(func(L *gopher.LState) int {
			close(h.signal(L.CheckString(1)))
			return 0
		}))
	})
	return h
}

func (h *hostOpTest) signal(name string) chan struct{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch, ok := h.signals[name]
	if !ok {
		ch = make(chan struct{})
		h.signals[name] = ch
	}
	return ch
}

// onExec runs fn on the executor with the session's Lua state and waits for it.
func (h *hostOpTest) onExec(fn func(L *gopher.LState)) {
	h.t.Helper()
	done := make(chan struct{})
	if !h.ref.Exec().Post(func() {
		defer close(done)
		fn(h.ref.Lua().Session().State)
	}) {
		h.t.Fatal("Post returned false")
	}
	waitFor(h.t, done, "executor")
}

// define installs a host operation on the executor.
func (h *hostOpTest) define(name string, deadline time.Duration, op HostOp) {
	h.onExec(func(*gopher.LState) { h.ref.Lua().DefineHostOp(name, deadline, op) })
}

// run executes a Lua chunk on the executor, failing the test on a Lua error.
func (h *hostOpTest) run(code string) {
	h.t.Helper()
	var err error
	h.onExec(func(L *gopher.LState) { err = L.DoString(code) })
	if err != nil {
		h.t.Fatalf("Lua: %v", err)
	}
}

// eval returns tostring(expr), evaluated on the executor.
func (h *hostOpTest) eval(expr string) string {
	h.t.Helper()
	var out string
	var err error
	h.onExec(func(L *gopher.LState) {
		if err = L.DoString("return tostring(" + expr + ")"); err == nil {
			out = L.ToString(-1)
			L.Pop(1)
		}
	})
	if err != nil {
		h.t.Fatalf("Lua eval %s: %v", expr, err)
	}
	return out
}

func (h *hostOpTest) expect(expr, want string) {
	h.t.Helper()
	if got := h.eval(expr); got != want {
		h.t.Errorf("%s = %q, want %q", expr, got, want)
	}
}

func immediate(v any, err error) func(context.Context) (any, error) {
	return func(context.Context) (any, error) { return v, err }
}

// Test: a call delivers its result to the fields and the callback
// (R205, R207, R208, R209, R211, R213, R215)
func TestHostOpDeliversResult(t *testing.T) {
	h := newHostOpTest(t)
	var opArgs int
	h.define("greet", 0, func(args []gopher.LValue) (func(context.Context) (any, error), error) {
		opArgs = len(args)
		return immediate("hello "+args[0].String(), nil), nil
	})
	h.run(`t = {msgError = "stale"}
		ui.greet(t, "msg", "bob", function(r, e) cbR, cbE = r, e; signal("greet") end)`)
	waitFor(t, h.signal("greet"), "greet callback")
	barrier(t, h.ref)

	if opArgs != 1 {
		t.Errorf("op saw %d arguments, want 1 (the callback must not be passed)", opArgs)
	}
	h.expect("t.msg", "hello bob")
	h.expect("t.msgError", "nil")
	h.expect("cbR", "hello bob")
	h.expect("cbE", "nil")
	if n := len(h.ref.Exec().PendingTailors()); n != 0 {
		t.Errorf("PendingTailors lists %d after delivery, want 0", n)
	}
}

// Test: delivery runs change detection (R212)
func TestHostOpDeliveryRunsChangeDetection(t *testing.T) {
	h := newHostOpTest(t)
	attachBrowser(h.s, h.id)
	h.define("echo", 0, func(args []gopher.LValue) (func(context.Context) (any, error), error) {
		return immediate(args[0], nil), nil
	})
	var batches atomic.Int32
	h.s.wsEndpoint.SetAfterBatch(func(string, bool) { batches.Add(1) })

	before := batches.Load() // before the call: delivery can finish before run returns
	h.run(`t = {} ui.echo(t, "v", 1, function() signal("echo") end)`)
	waitFor(t, h.signal("echo"), "echo callback")
	barrier(t, h.ref)
	if batches.Load() <= before {
		t.Errorf("delivery ran no change detection (afterBatch %d before, %d after)", before, batches.Load())
	}
}

// Test: a failing work clears the result and sets the error (R214, R215)
func TestHostOpFailureClearsResult(t *testing.T) {
	h := newHostOpTest(t)
	h.define("fails", 0, func([]gopher.LValue) (func(context.Context) (any, error), error) {
		return immediate(nil, errors.New("no luck")), nil
	})
	h.run(`t = {out = "stale"}
		ui.fails(t, "out", function(r, e) cbR, cbE = r, e; signal("fails") end)`)
	waitFor(t, h.signal("fails"), "fails callback")

	h.expect("t.out", "nil")
	h.expect("t.outError", "no luck")
	h.expect("cbR", "nil")
	h.expect("cbE", "no luck")
}

// Test: an op error is delivered without running work (R210)
func TestHostOpOpErrorSkipsWork(t *testing.T) {
	h := newHostOpTest(t)
	var ran atomic.Bool
	h.define("refuses", 0, func([]gopher.LValue) (func(context.Context) (any, error), error) {
		return func(context.Context) (any, error) { ran.Store(true); return nil, nil }, errors.New("bad args")
	})
	h.run(`t = {} ui.refuses(t, "x", function() signal("refuses") end)`)
	waitFor(t, h.signal("refuses"), "refuses callback")

	h.expect("t.xError", "bad args")
	if ran.Load() {
		t.Error("work ran although op returned an error")
	}
}

// Test: bad target or field raise at the call (R206)
func TestHostOpBadArgumentsRaise(t *testing.T) {
	h := newHostOpTest(t)
	h.define("greet", 0, func([]gopher.LValue) (func(context.Context) (any, error), error) {
		return immediate("x", nil), nil
	})
	h.run(`okTarget = pcall(ui.greet, "not a table", "msg")
		okField = pcall(ui.greet, {}, 42)`)
	h.expect("okTarget", "false")
	h.expect("okField", "false")
}

// Test: the call returns before the work finishes (R207)
func TestHostOpReturnsAtOnce(t *testing.T) {
	h := newHostOpTest(t)
	release := make(chan struct{})
	h.define("slow", 0, func([]gopher.LValue) (func(context.Context) (any, error), error) {
		return func(context.Context) (any, error) { <-release; return "done", nil }, nil
	})
	returned := make(chan struct{})
	h.ref.Exec().Post(func() {
		L := h.ref.Lua().Session().State
		if err := L.DoString(`t = {} ui.slow(t, "v", function() signal("slow") end)`); err != nil {
			t.Errorf("Lua: %v", err)
		}
		close(returned)
	})
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		close(release) // let the wedged executor go so the test can end
		t.Fatal("the host operation call did not return while its work was running")
	}
	close(release)
	waitFor(t, h.signal("slow"), "slow callback")
	h.expect("t.v", "done")
}

// Test: the deadline reaches work's context (R211, R216)
func TestHostOpDeadlineReachesWork(t *testing.T) {
	h := newHostOpTest(t)
	h.define("waits", 20*time.Millisecond, func([]gopher.LValue) (func(context.Context) (any, error), error) {
		return func(ctx context.Context) (any, error) { <-ctx.Done(); return nil, ctx.Err() }, nil
	})
	h.run(`t = {} ui.waits(t, "v", function() signal("waits") end)`)
	waitFor(t, h.signal("waits"), "waits callback")
	h.expect("t.vError", context.DeadlineExceeded.Error())
}

// Test: a work that ignores its context is bounded and delivers nothing (R216)
func TestHostOpBackstopBoundsStuckWork(t *testing.T) {
	h := newHostOpTest(t)
	release := make(chan struct{})
	finished := make(chan struct{})
	h.define("stuck", 20*time.Millisecond, func([]gopher.LValue) (func(context.Context) (any, error), error) {
		return func(context.Context) (any, error) {
			<-release
			defer close(finished)
			return "late", nil
		}, nil
	})
	h.run(`t = {} ui.stuck(t, "v")`)

	deadline := time.Now().Add(time.Second)
	for len(h.ref.Exec().PendingTailors()) != 0 {
		if time.Now().After(deadline) {
			close(release)
			t.Fatal("the stuck operation's Tailor was never closed")
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(release)
	waitFor(t, finished, "stuck work")
	time.Sleep(10 * time.Millisecond) // let the late EndUpdate be dropped
	barrier(t, h.ref)
	h.expect("t.v", "nil")
	h.expect("t.vError", "nil")
}
