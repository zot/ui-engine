// Tests for the Lua session's home on the Session (test-SessionRef.md, Item 8 of the
// coherency carve): the server keeps no table of Lua sessions; every lookup goes from
// the vended ID to the session and asks it.
// CRC: crc-Session.md, crc-SessionManager.md, crc-Server.md | Test: test-SessionRef.md | R230, R231, R232, R233, R234, R235
package server

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	gopher "github.com/yuin/gopher-lua"
)

// setMainLua replaces the test site's main.lua; sessions created afterwards run it.
func setMainLua(t *testing.T, s *Server, code string) {
	t.Helper()
	path := filepath.Join(s.config.Server.Dir, "lua", "main.lua")
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Test: the Lua session lives on the session (R230, R231, R232)
func TestLuaSessionLivesOnSession(t *testing.T) {
	s := newRefTestServer(t, nil)
	internalID, vendedID := newSession(t, s)
	ls := s.GetLuaSession(vendedID)
	if ls == nil {
		t.Fatal("GetLuaSession found nothing for a live session")
	}
	if got := s.sessions.Get(internalID).LuaSession(); got != ls {
		t.Fatal("GetLuaSession did not return the session's own Lua session")
	}
	if ref, ok := s.SessionRef(vendedID); !ok || !ref.Valid() {
		t.Fatal("no valid reference for a live session")
	}
}

// Test: a session can be found while main.lua runs (R230)
func TestSessionFoundWhileMainLuaRuns(t *testing.T) {
	s := newRefTestServer(t, nil)
	setMainLua(t, s, "session:setImmediate(function() immediateRan = true end)\n")
	_, vendedID := newSession(t, s)
	ref, ok := s.SessionRef(vendedID)
	if !ok {
		t.Fatal("no reference after creation")
	}
	ran := make(chan bool, 1)
	// Queued behind anything main.lua deferred, so it sees the outcome.
	ref.Exec().Post(func() {
		ran <- ref.Lua().Session().State.GetGlobal("immediateRan") == gopher.LTrue
	})
	select {
	case got := <-ran:
		if !got {
			t.Fatal("work main.lua deferred never ran: its session could not be found while main.lua ran")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the reading post never ran")
	}
}

// Test: a second attach panics (R230)
func TestSecondAttachPanics(t *testing.T) {
	s := newRefTestServer(t, nil)
	internalID, _ := newSession(t, s)
	sess := s.sessions.Get(internalID)
	defer func() {
		if recover() == nil {
			t.Fatal("a second attachLua did not panic")
		}
	}()
	sess.attachLua(sess.LuaSession())
}

// Test: a destroyed session cannot be found during teardown (R234)
func TestDestroyedSessionNotFoundDuringTeardown(t *testing.T) {
	s := newRefTestServer(t, nil)
	internalID, vendedID := newSession(t, s)
	var foundLua, foundRef bool
	s.sessions.SetOnSessionDestroyed(func(v string, sess *Session) {
		foundLua = s.GetLuaSession(v) != nil
		_, foundRef = s.SessionRef(v)
		s.DestroyLuaBackendForSession(v, sess)
	})
	s.sessions.DestroySession(internalID)
	if foundLua {
		t.Errorf("GetLuaSession(%s) found the session during its teardown (R234)", vendedID)
	}
	if foundRef {
		t.Errorf("SessionRef(%s) found the session during its teardown (R234)", vendedID)
	}
}

// Test: nothing is found after server shutdown (R232, R118)
func TestNothingFoundAfterShutdown(t *testing.T) {
	s := newRefTestServer(t, nil)
	_, first := newSession(t, s)
	_, second := newSession(t, s)
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{first, second} {
		if s.GetLuaSession(id) != nil {
			t.Errorf("GetLuaSession(%s) found a Lua session after Shutdown", id)
		}
		if _, ok := s.SessionRef(id); ok {
			t.Errorf("SessionRef(%s) succeeded after Shutdown", id)
		}
	}
	if ids := s.GetSessionIDs(); len(ids) != 0 {
		t.Errorf("GetSessionIDs after Shutdown = %v, want none", ids)
	}
}

// Test: sessions are enumerated from the registry (R233)
func TestSessionsEnumeratedFromRegistry(t *testing.T) {
	s := newRefTestServer(t, nil)
	var internals, vended []string
	for i := 0; i < 3; i++ {
		in, v := newSession(t, s)
		internals = append(internals, in)
		vended = append(vended, v)
	}
	check := func(want []string) {
		t.Helper()
		got := s.GetSessionIDs()
		sort.Strings(got)
		sort.Strings(want)
		if len(got) != len(want) {
			t.Fatalf("GetSessionIDs = %v, want %v", got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("GetSessionIDs = %v, want %v", got, want)
			}
		}
	}
	check(append([]string(nil), vended...))
	s.sessions.DestroySession(internals[1])
	check([]string{vended[0], vended[2]})
}

// Test: ExecuteInSessionAsync is an active post (R235)
func TestExecuteInSessionAsyncIsActive(t *testing.T) {
	s := newRefTestServer(t, nil)
	internalID, vendedID := newSession(t, s)
	attachBrowser(s, internalID)
	var batches atomic.Int32
	s.wsEndpoint.SetAfterBatch(func(string, bool) { batches.Add(1) })
	ref, _ := s.SessionRef(vendedID)

	var ran atomic.Bool
	s.ExecuteInSessionAsync(vendedID, func() (interface{}, error) {
		ran.Store(true)
		return nil, nil
	})
	done := make(chan struct{})
	ref.Exec().Post(func() { close(done) })
	waitFor(t, done, "barrier after ExecuteInSessionAsync")
	if !ran.Load() {
		t.Fatal("ExecuteInSessionAsync did not run its work")
	}
	if n := batches.Load(); n != 1 {
		t.Fatalf("afterBatch ran %d times after ExecuteInSessionAsync, want 1 (R235)", n)
	}

	unknownRan := false
	s.ExecuteInSessionAsync("no-such-session", func() (interface{}, error) {
		unknownRan = true
		return nil, nil
	})
	if unknownRan {
		t.Fatal("ExecuteInSessionAsync ran work for an unknown session")
	}
}
