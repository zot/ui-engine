// CRC: crc-Tailor.md | Test: test-Tailor.md
package server

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// barrier waits until everything already posted to the session executor has run.
func barrier(t *testing.T, ref SessionRef) {
	t.Helper()
	done := make(chan struct{})
	if !ref.Exec().Post(func() { close(done) }) {
		t.Fatal("barrier: Post returned false")
	}
	waitFor(t, done, "executor barrier")
}

func newTailorTest(t *testing.T) (*Server, SessionRef, string) {
	t.Helper()
	s := newRefTestServer(t, nil)
	internalID, vendedID := newSession(t, s)
	ref, ok := s.SessionRef(vendedID)
	if !ok {
		t.Fatal("no ref for new session")
	}
	return s, ref, internalID
}

func expectPanic(t *testing.T, what string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s did not panic", what)
		}
	}()
	fn()
}

var errStitch = errors.New("stitch failed on purpose")

// Test: a sewing runs its stitches in order and closes (R176, R177, R183, R191)
func TestTailorSewingRunsInOrderAndCloses(t *testing.T) {
	_, ref, _ := newTailorTest(t)
	tl := ref.Exec().Tailor("in-order", 0)
	var order []string
	done := make(chan struct{})
	tl.Sew(func() error {
		if tl.State() == nil {
			t.Error("State is nil inside a stitch")
		}
		order = append(order, "a")
		return nil
	}, false)
	tl.End(func() error {
		order = append(order, "b")
		close(done)
		return nil
	})
	waitFor(t, done, "End stitch")
	barrier(t, ref)

	if strings.Join(order, "") != "ab" {
		t.Fatalf("stitches ran as %v, want [a b]", order)
	}
	if tl.IsValid() {
		t.Error("Tailor still valid after End")
	}
	if n := len(ref.Exec().PendingTailors()); n != 0 {
		t.Errorf("PendingTailors lists %d after End, want 0", n)
	}
	if tl.Context().Err() == nil {
		t.Error("Context not cancelled after End")
	}
}

// Test: an error ends the sewing (R181)
func TestTailorErrorEndsSewing(t *testing.T) {
	_, ref, _ := newTailorTest(t)
	buf := captureLog(t)
	tl := ref.Exec().Tailor("fails", 0)
	var ranAfter atomic.Bool
	tl.Sew(func() error { return errStitch }, false)
	tl.Sew(func() error { ranAfter.Store(true); return nil }, false)
	barrier(t, ref)

	if ranAfter.Load() {
		t.Fatal("a stitch queued behind a failed stitch ran")
	}
	if tl.IsValid() {
		t.Error("Tailor still valid after a failed stitch")
	}
	if !strings.Contains(buf.String(), `Tailor "fails" failed`) {
		t.Errorf("failure not logged with the Tailor's name:\n%s", buf.String())
	}
}

// Test: a panic counts as an error (R180)
func TestTailorPanicCountsAsError(t *testing.T) {
	_, ref, _ := newTailorTest(t)
	tl := ref.Exec().Tailor("panics", 0)
	tl.Sew(func() error { panic("boom") }, false)
	barrier(t, ref)
	if tl.IsValid() {
		t.Fatal("Tailor still valid after a panicking stitch")
	}
}

// Test: updateOnErr runs change detection only on error (R182)
func TestTailorUpdateOnErr(t *testing.T) {
	s, ref, internalID := newTailorTest(t)
	attachBrowser(s, internalID)
	var batches atomic.Int32
	s.wsEndpoint.SetAfterBatch(func(string, bool) { batches.Add(1) })

	ref.Exec().Tailor("quiet", 0).Sew(func() error { return errStitch }, false)
	barrier(t, ref)
	if n := batches.Load(); n != 0 {
		t.Fatalf("afterBatch ran %d times for an error without updateOnErr, want 0", n)
	}

	ref.Exec().Tailor("loud", 0).Sew(func() error { return errStitch }, true)
	barrier(t, ref)
	if n := batches.Load(); n != 1 {
		t.Fatalf("afterBatch ran %d times for an error with updateOnErr, want 1", n)
	}
}

// Test: posting after End panics (R184)
func TestTailorPostAfterEndPanics(t *testing.T) {
	_, ref, _ := newTailorTest(t)
	tl := ref.Exec().Tailor("ended", 0)
	tl.End(func() error { return nil })
	expectPanic(t, "Sew after End", func() { tl.Sew(func() error { return nil }, false) })
}

// Test: posting after a failure logs once and drops (R185)
func TestTailorLatePostLogsOnceAndDrops(t *testing.T) {
	_, ref, _ := newTailorTest(t)
	buf := captureLog(t)
	tl := ref.Exec().Tailor("late", 0)
	tl.Sew(func() error { return errStitch }, false)
	barrier(t, ref)

	var ran atomic.Int32
	tl.Sew(func() error { ran.Add(1); return nil }, false)
	tl.Sew(func() error { ran.Add(1); return nil }, false)
	barrier(t, ref)

	if n := ran.Load(); n != 0 {
		t.Fatalf("%d stitches ran after the Tailor failed, want 0", n)
	}
	if n := strings.Count(buf.String(), `Tailor "late": stitch posted after it closed`); n != 1 {
		t.Fatalf("late-post logged %d times, want 1:\n%s", n, buf.String())
	}
}

// Test: State is only available inside a stitch (R187)
func TestTailorStateOutsideStitchPanics(t *testing.T) {
	_, ref, _ := newTailorTest(t)
	tl := ref.Exec().Tailor("state", 0)
	expectPanic(t, "State outside a stitch", func() { tl.State() })
}

// Test: the deadline fails the Tailor and cancels its context (R175, R189, R190)
func TestTailorDeadline(t *testing.T) {
	_, ref, _ := newTailorTest(t)
	short := ref.Exec().Tailor("short", 20*time.Millisecond)
	long := ref.Exec().Tailor("forever", 0)

	select {
	case <-short.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("deadline passed without cancelling the context")
	}
	if short.IsValid() {
		t.Error("Tailor still valid after its deadline")
	}
	if !long.IsValid() {
		t.Error("long-lived Tailor expired")
	}
	infos := ref.Exec().PendingTailors()
	if len(infos) != 1 || infos[0].Name != "forever" || !infos[0].LongLived {
		t.Errorf("PendingTailors = %+v, want only the long-lived Tailor", infos)
	}
}

// Test: destroying the session closes its Tailors (R193)
func TestTailorSessionDestroyClosesTailors(t *testing.T) {
	s, ref, internalID := newTailorTest(t)
	timed := ref.Exec().Tailor("timed", time.Hour)
	forever := ref.Exec().Tailor("forever", 0)
	if err := s.sessions.DestroySession(internalID); err != nil {
		t.Fatal(err)
	}
	for _, tl := range []*Tailor{timed, forever} {
		select {
		case <-tl.Context().Done():
		case <-time.After(time.Second):
			t.Fatalf("Tailor %q context not cancelled by session destroy", tl.name)
		}
		if tl.IsValid() {
			t.Errorf("Tailor %q still valid after session destroy", tl.name)
		}
	}
}

// Test: posting on an ended session closes the Tailor (R179, R186)
func TestTailorPostOnEndedSession(t *testing.T) {
	s, ref, _ := newTailorTest(t)
	tl := ref.Exec().Tailor("orphan", 0)
	s.GetLuaSession(ref.ID()).Shutdown() // no destroy sweep
	var ran atomic.Bool
	tl.Sew(func() error { ran.Store(true); return nil }, false)
	time.Sleep(50 * time.Millisecond)
	if ran.Load() {
		t.Fatal("stitch ran on an ended session")
	}
	if tl.IsValid() {
		t.Fatal("Tailor still valid after posting to an ended session")
	}
}

// Test: the pending list reports the last calling site (R178, R192)
func TestTailorPendingReportsLastSite(t *testing.T) {
	_, ref, _ := newTailorTest(t)
	tl := ref.Exec().Tailor("reporting", 0)
	tl.Sew(func() error { return nil }, false)
	barrier(t, ref)
	infos := ref.Exec().PendingTailors()
	if len(infos) != 1 || infos[0].Name != "reporting" {
		t.Fatalf("PendingTailors = %+v, want the one Tailor", infos)
	}
	if !strings.Contains(infos[0].LastSite, "tailor_test.go:") {
		t.Fatalf("LastSite %q does not name the calling test file", infos[0].LastSite)
	}
}
