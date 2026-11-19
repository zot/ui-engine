// CRC: crc-ExecAsync.md | Test: test-ExecAsync.md
package server

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	gopher "github.com/yuin/gopher-lua"
)

const execHelperEnv = "UI_ENGINE_EXEC_HELPER"

// TestExecHelperProcess is the child for the execAsync cases, not a test: with
// execHelperEnv set it acts on the arguments after "--" and exits.
func TestExecHelperProcess(t *testing.T) {
	if os.Getenv(execHelperEnv) != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	switch args[0] {
	case "echo":
		for _, a := range args[1:] {
			fmt.Println(a)
		}
		os.Exit(0)
	case "fail":
		fmt.Fprint(os.Stderr, "boom")
		os.Exit(3)
	case "sleep":
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	os.Exit(99)
}

// execTest is a host-op session whose Lua has helper(...) building an argv
// that re-runs this test binary as TestExecHelperProcess.
func execTest(t *testing.T) *hostOpTest {
	t.Helper()
	t.Setenv(execHelperEnv, "1")
	h := newHostOpTest(t)
	h.onExec(func(L *gopher.LState) {
		L.SetGlobal("bin", gopher.LString(os.Args[0]))
	})
	h.run(`function helper(...)
		local argv = {bin, "-test.run=^TestExecHelperProcess$", "--"}
		for _, a in ipairs({...}) do argv[#argv + 1] = a end
		return argv
	end`)
	return h
}

// Test: every session has ui.execAsync before main.lua runs (R250)
func TestExecAsyncDefinedBeforeMainLua(t *testing.T) {
	s := newRefTestServer(t, nil)
	setMainLua(t, s, "mainSawExecAsync = type(ui.execAsync)\n")
	_, vendedID := newSession(t, s)
	ref, ok := s.SessionRef(vendedID)
	if !ok {
		t.Fatal("no reference after creation")
	}
	got := make(chan string, 1)
	ref.Exec().Post(func() {
		got <- ref.Lua().Session().State.GetGlobal("mainSawExecAsync").String()
	})
	select {
	case v := <-got:
		if v != "function" {
			t.Fatalf("main.lua saw type(ui.execAsync) = %s, want function (R250)", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the reading post never ran")
	}
}

// Test: stdout lands in the field and the callback (R251)
func TestExecAsyncDeliversStdout(t *testing.T) {
	h := execTest(t)
	h.run(`t = {} ui.execAsync(t, "out", helper("echo", "a b", "x>y"), function(out, err)
		cbOut, cbErr = out, err signal("done") end)`)
	waitFor(t, h.signal("done"), "the execAsync callback")
	want := "a b\nx>y\n"
	if got := strings.ReplaceAll(h.eval("t.out"), "\r\n", "\n"); got != want {
		t.Errorf("t.out = %q (error %s), want %q", got, h.eval("t.outError"), want)
	}
	h.expect("t.outError", "nil")
	if got := strings.ReplaceAll(h.eval("cbOut"), "\r\n", "\n"); got != want {
		t.Errorf("callback stdout = %q, want %q", got, want)
	}
	h.expect("cbErr", "nil")
}

// Test: a failing program delivers its exit status and stderr (R251)
func TestExecAsyncFailure(t *testing.T) {
	h := execTest(t)
	h.run(`t = {} ui.execAsync(t, "out", helper("fail"), function() signal("done") end)`)
	waitFor(t, h.signal("done"), "the execAsync callback")
	h.expect("t.out", "nil")
	if got := h.eval("t.outError"); !strings.Contains(got, "3") || !strings.Contains(got, "boom") {
		t.Errorf("t.outError = %q, want the exit status 3 and stderr boom", got)
	}
}

// Test: a program past its limit is killed and the call fails (R252)
func TestExecAsyncTimeout(t *testing.T) {
	h := execTest(t)
	h.run(`t = {} local argv = helper("sleep") argv.timeout = 0.2
		ui.execAsync(t, "out", argv, function() signal("done") end)`)
	select {
	case <-h.signal("done"):
	case <-time.After(10 * time.Second):
		t.Fatal("a 0.2-second limit did not end a 30-second program within 10 seconds (R252)")
	}
	h.expect("t.out", "nil")
	if got := h.eval("t.outError"); !strings.Contains(got, "timed out") {
		t.Errorf("t.outError = %q, want it to say the program timed out", got)
	}
}
