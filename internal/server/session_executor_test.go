package server

// Tests for the session executor's life with its session (test-SessionExecutor.md).
// The executor used to live in a map on WebSocketEndpoint, created on first use and
// never stopped, so every destroyed session leaked its goroutine. These count
// goroutines; settles and within are in svc_test.go.
//
// CRC: crc-SessionManager.md, crc-Session.md, crc-WebSocketEndpoint.md | Test: test-SessionExecutor.md | R224, R225, R226, R227, R228

import (
	"errors"
	"runtime"
	"testing"
	"time"
)

// Test: destroying a session ends its executor (R224, R225, R227)
func TestSessionExecutor_DestroyEndsExecutor(t *testing.T) {
	m := NewSessionManager(time.Hour)
	before := runtime.NumGoroutine()
	ids := make([]string, 20)
	for i := range ids {
		sess, _, err := m.CreateSession()
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = sess.ID
		SvcSync(sess.Executor(), func() (int, error) { return 0, nil })
	}
	for _, id := range ids {
		m.DestroySession(id)
	}
	settles(t, before)
}

// Test: the destroy callback still has the executor (R225)
func TestSessionExecutor_DestroyCallbackStillHasExecutor(t *testing.T) {
	m := NewSessionManager(time.Hour)
	ran := false
	var callbackErr error
	m.SetOnSessionDestroyed(func(vendedID string, sess *Session) {
		_, callbackErr = SvcSync(sess.Executor(), func() (int, error) { ran = true; return 0, nil })
	})
	sess, _, err := m.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	within(t, 2*time.Second, "DestroySession", func() { m.DestroySession(sess.ID) })
	if callbackErr != nil || !ran {
		t.Errorf("teardown could not use the executor: ran=%v err=%v (R225)", ran, callbackErr)
	}
}

// Test: a session destroys itself from inside its executor (R226)
func TestSessionExecutor_DestroyFromInside(t *testing.T) {
	m := NewSessionManager(time.Hour)
	ws := NewWebSocketEndpoint(nil, m, nil)
	before := runtime.NumGoroutine()
	sess, _, err := m.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	var val interface{}
	within(t, 2*time.Second, "a session destroying itself from its executor", func() {
		val, err = ws.ExecuteInSession(sess.ID, func() (interface{}, error) {
			m.DestroySession(sess.ID)
			return 7, nil
		})
	})
	if val != 7 || err != nil {
		t.Errorf("got (%v, %v), want (7, nil)", val, err)
	}
	if m.SessionExists(sess.ID) {
		t.Error("the session survived destroying itself")
	}
	settles(t, before)
}

// Test: a failed creation leaks no executor (R227)
func TestSessionExecutor_FailedCreationLeaksNothing(t *testing.T) {
	m := NewSessionManager(time.Hour)
	m.SetOnSessionCreated(func(vendedID string, sess *Session) error {
		return errors.New("refused")
	})
	before := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		if _, _, err := m.CreateSession(); err == nil {
			t.Fatal("CreateSession succeeded with a failing create callback")
		}
	}
	settles(t, before)
}

// Test: work for a destroyed session creates nothing and never blocks (R228)
func TestSessionExecutor_WorkForDestroyedSession(t *testing.T) {
	m := NewSessionManager(time.Hour)
	ws := NewWebSocketEndpoint(nil, m, nil)
	sess, _, err := m.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	m.DestroySession(sess.ID)
	before := runtime.NumGoroutine()
	ran := make(chan struct{}, 60)
	var errs int
	within(t, 2*time.Second, "posting to a destroyed session", func() {
		for i := 0; i < 20; i++ {
			if _, err := ws.ExecuteInSession(sess.ID, func() (interface{}, error) {
				ran <- struct{}{}
				return nil, nil
			}); err != nil {
				errs++
			}
			ws.ExecuteInSessionAsync(sess.ID, func() (interface{}, error) {
				ran <- struct{}{}
				return nil, nil
			})
			ws.ExecuteInSessionPassive(sess.ID, func() { ran <- struct{}{} })
		}
	})
	if errs != 20 {
		t.Errorf("ExecuteInSession on a destroyed session: %d of 20 returned an error", errs)
	}
	time.Sleep(20 * time.Millisecond)
	if n := len(ran); n != 0 {
		t.Errorf("%d closures ran for a destroyed session", n)
	}
	settles(t, before)
}
