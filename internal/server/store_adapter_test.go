// Tests for the variable store's adapter (test-LuaTrackerAdapter.md, Item 9a of the
// coherency carve): it holds no per-session state, every operation names its session,
// and variable IDs repeat across sessions.
// CRC: crc-LuaTrackerAdapter.md, crc-LuaBackend.md | Test: test-LuaTrackerAdapter.md | R239, R240, R241, R242, R243
package server

import (
	"testing"
	"time"

	gopher "github.com/yuin/gopher-lua"
	"github.com/zot/ui-engine/internal/backend"
	"github.com/zot/ui-engine/internal/lua"
)

// onExec runs fn on the session's executor with its Lua state and waits for it. Variable
// creation resolves through Lua, so like real callers it must run there.
func onExec(t *testing.T, s *Server, vendedID string, fn func(L *gopher.LState)) {
	t.Helper()
	ref, ok := s.SessionRef(vendedID)
	if !ok {
		t.Fatalf("no reference for session %s", vendedID)
	}
	done := make(chan struct{})
	ref.Exec().Post(func() {
		defer close(done)
		fn(ref.Lua().Session().State)
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("executor work never ran")
	}
}

// createVar creates a variable through the adapter on the session's executor.
func createVar(t *testing.T, s *Server, vendedID string, parent int64, props map[string]string) (int64, error) {
	t.Helper()
	var id int64
	var err error
	onExec(t, s, vendedID, func(L *gopher.LState) {
		id, err = s.storeAdapter.CreateVariable(vendedID, parent, L.NewTable(), props)
	})
	return id, err
}

// childVar creates variable id under the root directly in the session's tracker, on its
// executor. Creating a non-root server variable through the store panics in the tracker
// (gap O9), so these tests make their children here.
func childVar(t *testing.T, s *Server, vendedID string, id int64, props map[string]string) {
	t.Helper()
	onExec(t, s, vendedID, func(*gopher.LState) {
		if s.storeAdapter.GetTracker(vendedID).CreateVariableWithId(id, nil, 1, "", props) == nil {
			t.Errorf("variable %d already exists in session %s", id, vendedID)
		}
	})
}

// withRoot creates a session with a root variable (ID 1) of the given type, so server
// variables have a parent.
func withRoot(t *testing.T, s *Server, typ string) (string, string) {
	t.Helper()
	internalID, vendedID := newSession(t, s)
	id, err := createVar(t, s, vendedID, 0, map[string]string{"type": typ})
	if err != nil || id != 1 {
		t.Fatalf("root variable for session %s: id %d, err %v", vendedID, id, err)
	}
	return internalID, vendedID
}

// runLua runs code on the session's executor and returns what it left in global name.
func runLua(t *testing.T, s *Server, vendedID, code, name string) gopher.LValue {
	t.Helper()
	ref, ok := s.SessionRef(vendedID)
	if !ok {
		t.Fatalf("no reference for session %s", vendedID)
	}
	type result struct {
		v   gopher.LValue
		err error
	}
	out := make(chan result, 1)
	ref.Exec().Post(func() {
		L := ref.Lua().Session().State
		err := L.DoString(code)
		out <- result{L.GetGlobal(name), err}
	})
	select {
	case r := <-out:
		if r.err != nil {
			t.Fatalf("Lua: %v", r.err)
		}
		return r.v
	case <-time.After(2 * time.Second):
		t.Fatal("Lua code never ran")
		return nil
	}
}

// Test: a variable ID resolves in the session that names it (R241)
func TestAdapterVariableIDResolvesInItsSession(t *testing.T) {
	s := newRefTestServer(t, nil)
	_, a := withRoot(t, s, "A")
	_, b := withRoot(t, s, "B")
	childVar(t, s, a, -1, map[string]string{"x": "from-a"})
	childVar(t, s, b, -1, map[string]string{"x": "from-b"})
	for _, c := range []struct{ session, value string }{{a, "from-a"}, {b, "from-b"}} {
		if got, _ := s.storeAdapter.GetProperty(c.session, -1, "x"); got != c.value {
			t.Errorf("GetProperty(%s, -1, x) = %q, want %q", c.session, got, c.value)
		}
		_, props, ok := s.storeAdapter.Get(c.session, -1)
		if !ok || props["x"] != c.value {
			t.Errorf("Get(%s, -1) = %v, %v; want x=%q", c.session, props, ok, c.value)
		}
	}
}

// Test: server variable IDs are counted per session (R240, R45)
func TestAdapterServerIDsCountedPerSession(t *testing.T) {
	s := newRefTestServer(t, nil)
	_, a := withRoot(t, s, "A")
	_, b := withRoot(t, s, "B")
	var got []int64
	for _, session := range []string{a, b, a} {
		got = append(got, s.storeAdapter.GetBackend(session).NextServerVarID())
	}
	if got[0] != -1 || got[1] != -1 || got[2] != -2 {
		t.Fatalf("server variable IDs = %v, want [-1 -1 -2]", got)
	}
}

// Test: session:destroyVariable destroys the variable (R242, R243)
func TestAdapterLuaDestroyVariable(t *testing.T) {
	s := newRefTestServer(t, nil)
	_, v := withRoot(t, s, "A")
	childVar(t, s, v, -1, nil)
	runLua(t, s, v, `session:destroyVariable(-1)`, "session")
	if s.storeAdapter.GetTracker(v).GetVariable(-1) != nil {
		t.Fatal("variable -1 survived session:destroyVariable (R243)")
	}
}

// Test: the adapter keeps no copy of the backend (R239)
func TestAdapterKeepsNoBackendCopy(t *testing.T) {
	s := newRefTestServer(t, nil)
	internalID, v := newSession(t, s)
	fresh := backend.NewLuaBackend(s.config, v, &lua.LuaResolver{})
	s.sessions.Get(internalID).SetBackend(fresh)
	if s.storeAdapter.GetTracker(v) != fresh.GetTracker() {
		t.Fatal("the adapter did not read the backend the session holds now (R239)")
	}
}
