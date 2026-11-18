package server

// Unit tests for the closure actor's delivery contract (test-ChanSvc.md).
//
// These exist because the ordering guarantee was assumed for a long time, was in
// fact absent (the post spawned a goroutine to send, so two posts raced), and
// nothing failed. They are written to fail if the delivery path loses it again,
// not merely to show that it holds today.
//
// CRC: crc-ChanSvc.md | Test: test-ChanSvc.md | R195, R196, R197, R198, R199, R201

import (
	"sync"
	"testing"
	"time"
)

// startedSvc returns a running actor, stopped at the end of the test.
func startedSvc(t *testing.T) *ChanSvc {
	t.Helper()
	s := NewChanSvc()
	s.Start()
	t.Cleanup(s.Stop)
	return s
}

// recorder collects the order closures ran in. Appends happen inside closures,
// which the actor serializes, so the mutex guards only the test's own read.
type recorder struct {
	mu  sync.Mutex
	got []int
}

func (r *recorder) note(i int) {
	r.mu.Lock()
	r.got = append(r.got, i)
	r.mu.Unlock()
}

func (r *recorder) seq() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int(nil), r.got...)
}

// wantInOrder fails unless the recorder saw exactly 0..n-1 in that order. Only
// the head of a long sequence is printed: a reordering shows up there, and 500
// ints in a failure message do not help anyone read it.
func wantInOrder(t *testing.T, r *recorder, n int, what string) {
	t.Helper()
	got := r.seq()
	for i, v := range got {
		if v != i {
			t.Errorf("%s: got %v…", what, got[:min(12, len(got))])
			return
		}
	}
	if len(got) != n {
		t.Errorf("%s: %d closures ran, want %d", what, len(got), n)
	}
}

// Test: closures posted by one goroutine run in the order posted (R195, R196).
// A spawned sender per post makes this fail, which is the regression to catch.
// n is large so a racing sender fails reliably rather than occasionally.
func TestChanSvcOrderedDelivery(t *testing.T) {
	const n = 500
	s := startedSvc(t)
	var r recorder

	done := make(chan struct{})
	for i := 0; i < n; i++ {
		Svc(s, func() {
			r.note(i)
			if i == n-1 {
				close(done)
			}
		})
	}
	<-done

	wantInOrder(t, &r, n, "delivery order broken (R195)")
}

// Test: ordering spans fire-and-forget and synchronous posts (R195). Both enter
// by the same door, so a design that ordered only one kind would pass the test
// above and still be wrong.
func TestChanSvcOrderingSpansPostKinds(t *testing.T) {
	const n = 120
	s := startedSvc(t)
	var r recorder

	for i := 0; i < n; i++ {
		if i%2 == 0 {
			Svc(s, func() { r.note(i) })
			continue
		}
		// The synchronous post also settles everything queued before it, which
		// is itself part of the guarantee.
		_, _ = SvcSync(s, func() (struct{}, error) { r.note(i); return struct{}{}, nil })
	}

	wantInOrder(t, &r, n, "ordering does not span post kinds (R195)")
}

// Test: exactly one closure runs at a time (R197, R198). Work moved off the loop
// goroutine, so serialization is now held by the in-flight flag rather than by
// the loop being busy — the property most easily lost refactoring accept/launch/next.
func TestChanSvcOneClosureAtATime(t *testing.T) {
	const posters = 8
	const each = 25
	s := startedSvc(t)

	var mu sync.Mutex
	var inFlight, maxFlight int

	var wg sync.WaitGroup
	for p := 0; p < posters; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				_, _ = SvcSync(s, func() (struct{}, error) {
					mu.Lock()
					inFlight++
					if inFlight > maxFlight {
						maxFlight = inFlight
					}
					mu.Unlock()

					time.Sleep(time.Millisecond)

					mu.Lock()
					inFlight--
					mu.Unlock()
					return struct{}{}, nil
				})
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if maxFlight != 1 {
		t.Errorf("max concurrent closures = %d, want 1 (R197, R198)", maxFlight)
	}
}

// Test: a closure may post fire-and-forget without deadlocking (R201). This is
// the property the old spawned-sender arrangement was buying, and the redesign
// has to keep it: the closure runs off the loop, so the post queues behind it.
func TestChanSvcReentrantFireAndForget(t *testing.T) {
	// Built by hand, not with startedSvc: if the property is broken the loop is
	// wedged, and a cleanup Stop would hang the test instead of failing it.
	s := NewChanSvc()
	s.Start()
	var r recorder
	done := make(chan struct{})

	// Two posts from inside, not one: a single nested post fits in the one-slot
	// channel even when the loop is wrongly busy running the outer closure, so
	// only the second would block.
	Svc(s, func() {
		r.note(0)
		Svc(s, func() { r.note(1) }) // posted from inside a running closure
		Svc(s, func() {
			r.note(2)
			close(done)
		})
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a closure posting fire-and-forget from inside itself deadlocked (R201)")
	}
	s.Stop()
	wantInOrder(t, &r, 3, "inner closures ran out of order")
}

// Test: a burst far larger than the channel is absorbed in order (R199). The
// queue behind the doorway takes the backlog, and taking it must not reorder —
// draining out of order is the failure the previous arrangement actually had.
func TestChanSvcBurstLargerThanChannelStaysOrdered(t *testing.T) {
	const n = 200
	s := startedSvc(t)
	var r recorder

	// Hold the first closure so everything after it must queue.
	release := make(chan struct{})
	done := make(chan struct{})
	Svc(s, func() { <-release; r.note(0) })
	for i := 1; i < n; i++ {
		Svc(s, func() {
			r.note(i)
			if i == n-1 {
				close(done)
			}
		})
	}
	close(release)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("burst did not drain")
	}
	wantInOrder(t, &r, n, "backlog drained out of order (R199)")
}
