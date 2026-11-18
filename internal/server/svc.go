package server

// The closure actor, ported from ark's svc.go. See specs/closure-actor.md for the
// contract and design/crc-ChanSvc.md for the card.
// CRC: crc-ChanSvc.md | R195, R196, R197, R198, R199, R200, R201, R202, R203

// chanSvcSlots is the depth of every actor's channel, and it is 1 on purpose. The
// channel is a doorway, not a buffer: bursts are absorbed by the queue behind it, and
// the loop does only O(1) bookkeeping, so depth would buy a poster nothing but a
// skipped rendezvous. What depth would cost is diagnosis: a loop wrongly given real
// work would let the first N posts through, so the stall would surface later and
// somewhere else. One slot blocks the very next poster, at the site.
//
// It cannot deadlock: the loop blocks only on its own receive, and neither accept nor
// next can block, so it always returns to drain.
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
// CRC: crc-ChanSvc.md | R195, R196, R197
type ChanSvc struct {
	ch chan func() // a doorway, not the buffer (R199)

	// loop goroutine only:
	queue   []func()
	running bool
}

// NewChanSvc builds an actor without starting it.
// CRC: crc-ChanSvc.md
func NewChanSvc() *ChanSvc {
	return &ChanSvc{ch: make(chan func(), chanSvcSlots)}
}

// Start launches the loop goroutine. Every closure it receives is bookkeeping (accept
// or next), so the loop does no caller work and cannot be held.
// CRC: crc-ChanSvc.md | R197
func (s *ChanSvc) Start() {
	go func() {
		for fn := range s.ch {
			if fn == nil {
				return // the stop sentinel
			}
			fn()
		}
	}()
}

// Stop shuts the loop down once everything already posted has been handled. Closures
// still queued behind an in-flight one are abandoned. The channel is deliberately never
// closed: every closure posts next back when it finishes, and a closure finishing after
// a close would panic sending on a closed channel.
// CRC: crc-ChanSvc.md | R203
func (s *ChanSvc) Stop() { s.ch <- nil }

// post is the single door onto the loop. The send happens here, on the caller's
// goroutine, which is what makes delivery ordered.
// CRC: crc-ChanSvc.md | R195, R196
func (s *ChanSvc) post(fn func()) { s.ch <- func() { s.accept(fn) } }

// accept runs on the loop goroutine: queue the closure if one is in flight, otherwise
// start it.
// CRC: crc-ChanSvc.md | R197, R198
func (s *ChanSvc) accept(fn func()) {
	if s.running {
		s.queue = append(s.queue, fn)
		return
	}
	s.launch(fn)
}

// launch runs on the loop goroutine: mark the actor busy and run the closure on its own
// goroutine, which posts next back when it finishes.
//
// No recover: a panicking closure takes the process down. The state the actor protects
// is in an unknown condition once a closure has panicked, and a central recover that
// forgot to clear running would queue every later closure forever, silently.
// CRC: crc-ChanSvc.md | R197, R198, R200
func (s *ChanSvc) launch(fn func()) {
	s.running = true
	go func() {
		fn()
		s.ch <- s.next
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

// Svc posts a fire-and-forget closure. Safe from inside a running closure: that
// closure runs off the loop goroutine, so this merely queues behind it.
// CRC: crc-ChanSvc.md | R195, R201
func Svc(s *ChanSvc, fn func()) { s.post(fn) }

// SvcSync posts a closure and blocks until it completes. Never call it from inside a
// running closure: the nested post queues behind the running one, which is blocked here
// waiting for it.
// CRC: crc-ChanSvc.md | R195, R201
func SvcSync[T any](s *ChanSvc, fn func() (T, error)) (T, error) {
	done := make(chan struct{})
	var val T
	var err error
	s.post(func() {
		val, err = fn()
		close(done)
	})
	<-done
	return val, err
}
