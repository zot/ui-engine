// Tests for one executor per session (Item 14 of the coherency carve): a session's
// Lua state is touched only on its executor.
// CRC: crc-LuaSession.md, crc-Server.md, crc-SessionRef.md | Test: test-OneExecutor.md | R254, R255, R256, R260, R262
package server

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gopher "github.com/yuin/gopher-lua"
)

// Test: change detection never overlaps work posted from outside (R254, R256)
func TestChangeDetectionNeverOverlapsOutsideWork(t *testing.T) {
	s := newRefTestServer(t, nil)
	setMainLua(t, s, "App = session:prototype('App', {n = 0})\nfunction App:probe() probeHit() return self.n end\napp = session:create(App)\nsession:createAppVariable(app)\n")
	internalID, vendedID := newSession(t, s)
	s.wsEndpoint.register("probe-conn", nil, internalID) // a connection, so change detection runs
	ref, ok := s.SessionRef(vendedID)
	if !ok {
		t.Fatal("no reference")
	}
	ls := ref.Lua().Session()

	// probeHit runs inside App:probe(), the Lua method variable 2 is bound to. Change
	// detection evaluates it on the session executor.
	var inOutside, probeCalls, overlaps atomic.Int32
	onExec(t, s, vendedID, func(L *gopher.LState) {
		L.SetGlobal("probeHit", L.NewFunction(func(*gopher.LState) int {
			probeCalls.Add(1)
			if inOutside.Load() > 0 {
				overlaps.Add(1)
			}
			return 0
		}))
		s.storeAdapter.GetTracker(vendedID).CreateVariableWithId(2, nil, 1, "probe()", nil)
	})

	const n = 100
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { // change detection on the session executor
		defer wg.Done()
		for i := 0; i < n; i++ {
			ref.Exec().PostUpdate(func() {})
			time.Sleep(time.Millisecond)
		}
	}()
	go func() { // the WithLua path: Lua work posted from outside
		defer wg.Done()
		for i := 0; i < n; i++ {
			ls.ExecuteInSession(vendedID, func() (interface{}, error) {
				inOutside.Add(1)
				defer inOutside.Add(-1)
				// Read only, so an overlap is counted rather than crashing the binary.
				_ = ls.State.GetField(ls.State.GetGlobal("app"), "n")
				time.Sleep(2 * time.Millisecond)
				return nil, nil
			})
		}
	}()
	wg.Wait()
	onExec(t, s, vendedID, func(*gopher.LState) {}) // drain what is still queued

	if probeCalls.Load() == 0 {
		t.Fatal("App:probe() never ran: change detection was not exercised, so this proves nothing")
	}
	if got := overlaps.Load(); got != 0 {
		t.Errorf("change detection ran Lua %d times while outside work was inside the Lua state (of %d probe calls)", got, probeCalls.Load())
	}
}

// Test: work already on the executor reaches Lua in place (R255)
func TestOnExecutorWorkRunsInPlace(t *testing.T) {
	s := newRefTestServer(t, nil)
	_, vendedID := newSession(t, s)
	ref, ok := s.SessionRef(vendedID)
	if !ok {
		t.Fatal("no reference")
	}
	set := func(name string) func() {
		return func() { ref.Lua().Session().State.SetGlobal(name, gopher.LTrue) }
	}
	ref.Exec().Post(set("viaPost"))
	ref.Exec().PostUpdate(set("viaPostUpdate"))
	go s.ExecuteInSession(vendedID, func() (interface{}, error) { set("viaExecute")(); return nil, nil })
	ref.Exec().Post(func() {
		ref.Lua().Session().State.DoString(`session:setImmediate(function() viaImmediate = true end)`)
	})

	deadline := time.Now().Add(2 * time.Second)
	for _, name := range []string{"viaPost", "viaPostUpdate", "viaExecute", "viaImmediate"} {
		for {
			got := make(chan bool, 1)
			go ref.Exec().Post(func() { got <- ref.Lua().Session().State.GetGlobal(name) == gopher.LTrue })
			var done bool
			select {
			case done = <-got:
			case <-time.After(time.Until(deadline)):
			}
			if done {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s never ran within 2 seconds: work on the executor waited for itself", name)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// Test: a session destroyed from inside its executor closes its VM only after (R260, R262)
func TestSelfDestroyClosesVMAfterDrain(t *testing.T) {
	s := newRefTestServer(t, nil)
	internalID, vendedID := newSession(t, s)
	ref, ok := s.SessionRef(vendedID)
	if !ok {
		t.Fatal("no reference")
	}
	ls := ref.Lua().Session()

	readWorked := make(chan bool, 1)
	ref.Exec().Post(func() {
		if err := s.sessions.DestroySession(internalID); err != nil {
			t.Errorf("DestroySession: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
		func() {
			defer func() {
				if recover() != nil {
					readWorked <- false
				}
			}()
			readWorked <- ls.State.DoString("stillOpen = true") == nil
		}()
	})

	select {
	case ok := <-readWorked:
		if !ok {
			t.Fatal("the VM was closed under the closure that destroyed its own session")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the destroying closure never finished")
	}
	deadline := time.Now().Add(2 * time.Second)
	for !ls.VMClosed() {
		if time.Now().After(deadline) {
			t.Fatal("the VM was never closed after the executor drained")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
