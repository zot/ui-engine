package server

// The closure actor, ported from ark's svc.go. See specs/closure-actor.md for the
// contract and design/crc-ChanSvc.md for the card.
// CRC: crc-ChanSvc.md | R195, R196, R197, R198, R199, R200, R201, R202, R218, R219, R220, R221, R222, R223

import "errors"

// ErrStopped is what a synchronous post to a stopped actor returns, and what a
// synchronous caller gets when its closure was abandoned by a stop.
// CRC: crc-ChanSvc.md | R220, R221
var ErrStopped = errors.New("executor stopped")

// chanSvcSlots is the depth of every actor's channel, and it is 1 on purpose. The
// channel is a doorway, not a buffer: bursts are absorbed by the queue behind it, and
// the loop does only O(1) bookkeeping, so depth would buy a poster nothing but a
// skipped rendezvous. What depth would cost is diagnosis: a loop wrongly given real
// work would let the first N posts through, so the stall would surface later and
// somewhere else. One slot blocks the very next poster, at the site.
//
// It cannot deadlock: the loop blocks only on its own receive, and neither accept nor
// next can block, so it always returns to drain. Once the loop has stopped, nobody
// drains, which is why every send selects on done.
// CRC: crc-ChanSvc.md | R199
const chanSvcSlots = 1

// ChanSvc serializes the work posted to it: exactly one closure runs at any instant,
// and closures posted by one goroutine run in the order that goroutine posted them.
//
// The order guarantee rests on two things that hold each other up. A post is a direct
// send on the calling goroutine; handing the send to a spawned goroutine lets two
// posts race, which is what ui-engine did until 2026-11-18. That is only viable
// because the loop goroutine never runs a caller's closure: it manipulates the queue
// and the in-flight flag and nothing else, so it is effectively always back at the
// receive.
//
// queue and running are touched only on the loop goroutine, so neither needs a lock.
// CRC: crc-ChanSvc.md | R195, R196, R197, R219
type ChanSvc struct {
	ch   chan func()   // a doorway, not the buffer (R199)
	done chan struct{} // closed by the loop, and only by it, when it stops (R219)

	// loop goroutine only:
	queue   []*job
	running bool
}

// job is one posted closure. launched is written only on the loop goroutine, before
// that goroutine closes done, so a reader woken by done reads it without a race.
// CRC: crc-ChanSvc.md | R218, R221
type job struct {
	run      func()
	launched bool
}

// NewChanSvc builds an actor without starting it.
// CRC: crc-ChanSvc.md
func NewChanSvc() *ChanSvc {
	return &ChanSvc{ch: make(chan func(), chanSvcSlots), done: make(chan struct{})}
}

// Start launches the loop goroutine. Every closure it receives is bookkeeping (accept
// or next), so the loop does no caller work and cannot be held. On the stop sentinel it
// closes done and returns, abandoning whatever is still queued.
// CRC: crc-ChanSvc.md | R197, R218, R219
func (s *ChanSvc) Start() {
	go func() {
		for fn := range s.ch {
			if fn == nil {
				close(s.done) // the stop sentinel
				return
			}
			fn()
		}
	}()
}

// Stop shuts the loop down once everything already posted has been handled. A closure
// in flight runs to completion; closures still queued behind it are abandoned. The
// sentinel goes through send, so a Stop after the loop has ended returns at once, and a
// second sentinel sent before the loop took the first lies unread in the slot: Stop is
// idempotent with no flag of its own. Calling it from inside a running closure is
// fine, since closures run off the loop goroutine.
// CRC: crc-ChanSvc.md | R218, R223
func (s *ChanSvc) Stop() { s.send(nil) }

// send is the one way anything enters the channel. Once the actor has stopped it
// returns false at once instead of blocking on a channel nobody drains.
// CRC: crc-ChanSvc.md | R219, R222
func (s *ChanSvc) send(fn func()) bool {
	select {
	case s.ch <- fn:
		return true
	case <-s.done:
		return false
	}
}

// post is the single door onto the loop. The send happens here, on the caller's
// goroutine, which is what makes delivery ordered.
// CRC: crc-ChanSvc.md | R195, R196, R219
func (s *ChanSvc) post(j *job) bool { return s.send(func() { s.accept(j) }) }

// accept runs on the loop goroutine: queue the job if one is in flight, otherwise
// start it.
// CRC: crc-ChanSvc.md | R197, R198
func (s *ChanSvc) accept(j *job) {
	if s.running {
		s.queue = append(s.queue, j)
		return
	}
	s.launch(j)
}

// launch runs on the loop goroutine: mark the job launched and the actor busy, and run
// the closure on its own goroutine, which reports back when it finishes. After a stop
// that report is a no-op.
//
// No recover: a panicking closure takes the process down. The state the actor protects
// is in an unknown condition once a closure has panicked, and a central recover that
// forgot to clear running would queue every later closure forever, silently.
// CRC: crc-ChanSvc.md | R197, R198, R200, R218, R222
func (s *ChanSvc) launch(j *job) {
	j.launched = true
	s.running = true
	go func() {
		j.run()
		s.send(s.next)
	}()
}

// next runs on the loop goroutine: start whatever is waiting, or go idle.
// CRC: crc-ChanSvc.md | R197, R202
func (s *ChanSvc) next() {
	if len(s.queue) == 0 {
		s.running = false
		return
	}
	s.launch(popFront(&s.queue))
}

// popFront removes and returns the queue's head, clearing its slot first so a run
// closure and everything it captured stop being reachable through the backing array.
// CRC: crc-ChanSvc.md | R202
func popFront[T any](q *[]T) T {
	head := (*q)[0]
	var zero T
	(*q)[0] = zero
	*q = (*q)[1:]
	return head
}

// Svc posts a fire-and-forget closure; once the actor has stopped it is dropped. Safe
// from inside a running closure: that closure runs off the loop goroutine, so this
// merely queues behind it.
// CRC: crc-ChanSvc.md | R195, R201, R219
func Svc(s *ChanSvc, fn func()) { s.post(&job{run: fn}) }

// SvcSync posts a closure and blocks until it completes. On a stopped actor it returns
// ErrStopped at once. Woken by a stop while waiting, it keeps waiting when its closure
// is in flight, which runs to completion, and returns ErrStopped when the closure was
// abandoned. Never call it from inside a running closure: the nested post queues
// behind the running one, which is blocked here waiting for it.
// CRC: crc-ChanSvc.md | R195, R201, R218, R220, R221
func SvcSync[T any](s *ChanSvc, fn func() (T, error)) (T, error) {
	finished := make(chan struct{})
	var val T
	var err error
	j := &job{run: func() {
		val, err = fn()
		close(finished)
	}}
	if !s.post(j) {
		return val, ErrStopped
	}
	select {
	case <-finished:
	case <-s.done:
		if !j.launched {
			return val, ErrStopped
		}
		<-finished
	}
	return val, err
}
