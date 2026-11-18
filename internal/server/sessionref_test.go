// CRC: crc-SessionRef.md | Test: test-SessionRef.md
package server

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	gopher "github.com/yuin/gopher-lua"
	"github.com/zot/ui-engine/internal/config"
)

// newRefTestServer builds a Lua-enabled server over a minimal site directory.
func newRefTestServer(t *testing.T, mutate func(*config.Config)) *Server {
	t.Helper()
	site := t.TempDir()
	if err := os.MkdirAll(filepath.Join(site, "lua"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(site, "lua", "main.lua"), []byte("-- test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	cfg.Server.Dir = site
	cfg.Server.Socket = filepath.Join(t.TempDir(), "ui.sock")
	cfg.Lua.Hotload = false
	if mutate != nil {
		mutate(cfg)
	}
	return New(cfg)
}

// newSession creates a session and returns its internal and vended IDs.
func newSession(t *testing.T, s *Server) (string, string) {
	t.Helper()
	sess, vendedID, err := s.sessions.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	return sess.ID, vendedID
}

// attachBrowser makes the endpoint report a browser connection for the session.
func attachBrowser(s *Server, internalID string) {
	s.wsEndpoint.mu.Lock()
	s.wsEndpoint.sessionBindings["test-conn"] = internalID
	s.wsEndpoint.mu.Unlock()
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// Test: a reference resolves only its own session (R155, R156)
func TestSessionRefDoesNotFollowReusedVendedID(t *testing.T) {
	s := newRefTestServer(t, nil)
	internalID, vendedID := newSession(t, s)
	old, ok := s.SessionRef(vendedID)
	if !ok || !old.Valid() {
		t.Fatalf("fresh ref for %s not valid", vendedID)
	}

	if err := s.sessions.DestroySession(internalID); err != nil {
		t.Fatal(err)
	}
	if old.Valid() {
		t.Fatal("ref still valid after its session was destroyed")
	}

	_, reused := newSession(t, s)
	if reused != vendedID {
		t.Fatalf("expected the vended ID %s to be reused, got %s", vendedID, reused)
	}
	if old.Valid() {
		t.Fatalf("old ref reports valid against the new session %s", reused)
	}
	if old.Exec().Post(func() {}) {
		t.Fatal("Post on the old ref returned true")
	}
	fresh, ok := s.SessionRef(reused)
	if !ok || !fresh.Valid() {
		t.Fatal("fresh ref for the reused ID not valid")
	}
}

// Test: Post runs with the Lua context and without change detection (R159, R160)
func TestPostIsPassivePostUpdateIsActive(t *testing.T) {
	s := newRefTestServer(t, nil)
	internalID, vendedID := newSession(t, s)
	attachBrowser(s, internalID)
	var batches atomic.Int32
	s.wsEndpoint.SetAfterBatch(func(string, bool) { batches.Add(1) })
	ref, _ := s.SessionRef(vendedID)

	sawSession := make(chan bool, 1)
	if !ref.Exec().Post(func() {
		ls := ref.Lua().Session()
		sawSession <- ls != nil && ls.State.GetGlobal("session") != gopher.LNil
	}) {
		t.Fatal("Post returned false")
	}
	barrier := make(chan struct{})
	ref.Exec().Post(func() { close(barrier) })
	waitFor(t, barrier, "passive barrier")
	if !<-sawSession {
		t.Error("posted work did not see the session global")
	}
	if n := batches.Load(); n != 0 {
		t.Fatalf("afterBatch ran %d times after Post, want 0", n)
	}

	done := make(chan struct{})
	if !ref.Exec().PostUpdate(func() {}) {
		t.Fatal("PostUpdate returned false")
	}
	ref.Exec().Post(func() { close(done) })
	waitFor(t, done, "active barrier")
	if n := batches.Load(); n != 1 {
		t.Fatalf("afterBatch ran %d times after PostUpdate, want 1", n)
	}
}

// Test: posting on an invalid reference does nothing (R161)
func TestPostOnInvalidRefQueuesNothing(t *testing.T) {
	s := newRefTestServer(t, nil)
	internalID, vendedID := newSession(t, s)
	ref, _ := s.SessionRef(vendedID)
	if err := s.sessions.DestroySession(internalID); err != nil {
		t.Fatal(err)
	}
	ran := make(chan struct{})
	if ref.Exec().Post(func() { close(ran) }) {
		t.Fatal("Post on a destroyed session returned true")
	}
	if ref.Exec().PostUpdate(func() { close(ran) }) {
		t.Fatal("PostUpdate on a destroyed session returned true")
	}
	select {
	case <-ran:
		t.Fatal("work posted to a destroyed session ran")
	case <-time.After(100 * time.Millisecond):
	}
}

// Test: views and station round-trip (R157, R158)
func TestViewsAndStationRoundTrip(t *testing.T) {
	s := newRefTestServer(t, nil)
	_, vendedID := newSession(t, s)
	ref, _ := s.SessionRef(vendedID)
	for name, got := range map[string]SessionRef{
		"Lua":     ref.Lua().Ref(),
		"Browser": ref.Browser().Ref(),
		"Exec":    ref.Exec().Ref(),
		"station": ref.Exec().Station().Lua.Ref(),
	} {
		if got.ID() != vendedID {
			t.Errorf("%s view returned ref %q, want %q", name, got.ID(), vendedID)
		}
	}
	if _, ok := s.SessionRef("no-such-session"); ok {
		t.Error("SessionRef found a session that does not exist")
	}
}

// Test: identity alone invalidates a replaced session (R155)
func TestSessionRefChecksIdentity(t *testing.T) {
	s := newRefTestServer(t, nil)
	_, first := newSession(t, s)
	_, second := newSession(t, s)
	ref, _ := s.SessionRef(first)

	s.luaSessionsMu.Lock()
	original := s.luaSessions[first]
	s.luaSessions[first] = s.luaSessions[second] // replaced, but not shut down
	s.luaSessionsMu.Unlock()
	t.Cleanup(func() {
		s.luaSessionsMu.Lock()
		s.luaSessions[first] = original
		s.luaSessionsMu.Unlock()
	})

	if original.IsShutdown() {
		t.Fatal("setup: the original Lua session should still be running")
	}
	if ref.Valid() {
		t.Fatal("ref still valid after another Lua session took its vended ID")
	}
}

// Test: shutdown alone invalidates a reference (R155)
func TestSessionRefChecksShutdown(t *testing.T) {
	s := newRefTestServer(t, nil)
	_, vendedID := newSession(t, s)
	ref, _ := s.SessionRef(vendedID)
	s.GetLuaSession(vendedID).Shutdown() // still registered, as during a server shutdown
	if ref.Valid() {
		t.Fatal("ref still valid after its Lua session shut down")
	}
}

// Test: the browser view reports connection and pushes on demand (R162, R163)
func TestBrowserViewConnectedAndUpdate(t *testing.T) {
	s := newRefTestServer(t, nil)
	internalID, vendedID := newSession(t, s)
	var batches atomic.Int32
	s.wsEndpoint.SetAfterBatch(func(string, bool) { batches.Add(1) })
	ref, _ := s.SessionRef(vendedID)

	if ref.Browser().Connected() {
		t.Fatal("Connected is true with no browser attached")
	}
	attachBrowser(s, internalID)
	if !ref.Browser().Connected() {
		t.Fatal("Connected is false with a browser attached")
	}

	done := make(chan struct{})
	ref.Exec().Post(func() {
		ref.Browser().Update()
		close(done)
	})
	waitFor(t, done, "passive post calling Update")
	if n := batches.Load(); n != 1 {
		t.Fatalf("afterBatch ran %d times after Update in a passive post, want 1", n)
	}
}
