package server

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	gopher "github.com/yuin/gopher-lua"
)

// Tailor carries one sewing operation: stitches posted to the session executor,
// the operation's state, and its place on the session's pending list. It exists
// for one operation and then disappears.
// CRC: crc-Tailor.md | R174, R191, R194
type Tailor struct {
	ref       SessionRef
	session   *Session
	name      string
	created   time.Time
	longLived bool
	ctx       context.Context
	cancel    context.CancelFunc

	mu         sync.Mutex
	timer      *time.Timer
	ended      bool // End or EndUpdate posted (R184)
	closed     bool // posts nothing more (R183, R185)
	failed     bool
	warnedLate bool // a post after closing has been logged (R185)
	lastStitch time.Time
	lastSite   string

	inStitch atomic.Bool    // one of this Tailor's stitches is running (R187)
	state    *gopher.LState // set only while inStitch, on the executor
}

// TailorInfo is a snapshot of one open Tailor.
// CRC: crc-Tailor.md | R192
type TailorInfo struct {
	Name       string
	Created    time.Time
	LastStitch time.Time
	LastSite   string // file:line of the last Sew, SewUpdate, End or EndUpdate call
	LongLived  bool
}

// Tailor creates a Tailor for one sewing operation on this session. A deadline
// of 0 makes it long-lived; otherwise it closes as failed when the deadline passes.
// CRC: crc-SessionRef.md | Seq: seq-tailor.md#1.1 | R174, R175, R189, R191
func (v ExecRig) Tailor(name string, deadline time.Duration) *Tailor {
	ctx, cancel := context.WithCancel(context.Background())
	t := &Tailor{
		ref:       v.ref,
		name:      name,
		created:   time.Now(),
		longLived: deadline == 0,
		ctx:       ctx,
		cancel:    cancel,
	}
	t.session = v.ref.session()
	if t.session == nil {
		t.close("session ended", true)
		return t
	}
	t.session.addTailor(t)
	if deadline > 0 {
		t.mu.Lock()
		t.timer = time.AfterFunc(deadline, func() { t.close("deadline exceeded", true) })
		t.mu.Unlock()
	}
	return t
}

// PendingTailors lists the session's open Tailors.
// CRC: crc-SessionRef.md | R192
func (v ExecRig) PendingTailors() []TailorInfo {
	sess := v.ref.session()
	if sess == nil {
		return nil
	}
	ts := sess.openTailors()
	infos := make([]TailorInfo, 0, len(ts))
	for _, t := range ts {
		infos = append(infos, t.info())
	}
	return infos
}

// session returns the reference's *Session, or nil when the reference is invalid.
// CRC: crc-SessionRef.md | R191
func (r SessionRef) session() *Session {
	_, internalID, ok := r.resolve()
	if !ok {
		return nil
	}
	sess, _ := r.srv.sessions.GetSession(internalID)
	return sess
}

// Sew posts a passive stitch. When fn errors, change detection runs afterward
// only if updateOnErr is set.
// CRC: crc-Tailor.md | R176, R182
func (t *Tailor) Sew(fn func() error, updateOnErr bool) { t.post(fn, false, false, updateOnErr) }

// SewUpdate posts an active stitch: change detection runs afterward.
// CRC: crc-Tailor.md | R176, R177
func (t *Tailor) SewUpdate(fn func() error) { t.post(fn, true, false, false) }

// End posts the passive terminal stitch; the Tailor closes after it runs.
// CRC: crc-Tailor.md | R176, R183
func (t *Tailor) End(fn func() error) { t.post(fn, false, true, false) }

// EndUpdate posts the active terminal stitch; the Tailor closes after it runs.
// CRC: crc-Tailor.md | R176, R177, R183
func (t *Tailor) EndUpdate(fn func() error) { t.post(fn, true, true, false) }

// post records the stitch and posts it without waiting. Posting after End is a
// programming error and panics; posting after any other close logs once and drops.
// CRC: crc-Tailor.md | Seq: seq-tailor.md#1.2 | R176, R178, R179, R184, R185
func (t *Tailor) post(fn func() error, active, terminal, updateOnErr bool) {
	site := callerSite(3) // the caller of Sew, SewUpdate, End or EndUpdate
	t.mu.Lock()
	if t.ended {
		t.mu.Unlock()
		panic(fmt.Sprintf("Tailor %q: stitch posted after End, from %s", t.name, site))
	}
	if t.closed {
		warn := !t.warnedLate
		t.warnedLate = true
		t.mu.Unlock()
		if warn {
			t.log("Tailor %q: stitch posted after it closed, from %s; dropped", t.name, site)
		}
		return
	}
	t.lastStitch = time.Now()
	t.lastSite = site
	if terminal {
		t.ended = true
	}
	t.mu.Unlock()

	run := func() { t.runStitch(fn, active, terminal, updateOnErr) }
	exec := t.ref.Exec()
	posted := false
	if active {
		posted = exec.PostUpdate(run)
	} else {
		posted = exec.Post(run)
	}
	if !posted {
		// Seq: seq-tailor.md#1.2.3
		t.close("session ended", true)
	}
}

// runStitch runs one stitch on the executor. A Tailor already closed by a
// failure, its deadline or its session ending runs nothing more.
// CRC: crc-Tailor.md | Seq: seq-tailor.md#1.3 | R180, R181, R182, R183, R187
func (t *Tailor) runStitch(fn func() error, active, terminal, updateOnErr bool) {
	if t.isClosed() {
		return
	}
	ls := t.ref.Lua().Session()
	if ls == nil {
		t.close("session ended", true)
		return
	}
	t.state = ls.State
	t.inStitch.Store(true)
	err := call(fn)
	t.inStitch.Store(false)
	t.state = nil

	if err != nil {
		// Seq: seq-tailor.md#1.3.3
		t.close(fmt.Sprintf("stitch failed: %v", err), true)
		if !active && updateOnErr {
			t.ref.Browser().Update()
		}
		return
	}
	if terminal {
		t.close("", false)
	}
}

// call runs fn, turning a panic into an error.
// CRC: crc-Tailor.md | R180
func call(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn()
}

// close ends the Tailor once: stop the deadline timer, cancel the context, leave
// the pending list, and log a failure with the name, age and last calling site.
// CRC: crc-Tailor.md | Seq: seq-tailor.md#2 | R189, R190, R191, R193
func (t *Tailor) close(reason string, failed bool) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.failed = failed
	timer, site := t.timer, t.lastSite
	t.mu.Unlock()

	if timer != nil {
		timer.Stop()
	}
	t.cancel()
	if t.session != nil {
		t.session.removeTailor(t)
	}
	if failed {
		t.log("Tailor %q failed after %s: %s (last stitch from %s)",
			t.name, time.Since(t.created).Round(time.Millisecond), reason, site)
	}
}

func (t *Tailor) isClosed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

func (t *Tailor) log(format string, args ...interface{}) {
	if t.ref.srv != nil {
		t.ref.srv.config.Log(0, format, args...)
	}
}

// IsValid reports whether the Tailor is open and its session is valid.
// CRC: crc-Tailor.md | R186
func (t *Tailor) IsValid() bool { return !t.isClosed() && t.ref.Valid() }

// State returns the session's Lua state while one of this Tailor's stitches is
// running, and panics otherwise. The Tailor never hands out the LuaSession.
// CRC: crc-Tailor.md | R187, R188
func (t *Tailor) State() *gopher.LState {
	if !t.inStitch.Load() {
		panic(fmt.Sprintf("Tailor %q: State called outside a running stitch", t.name))
	}
	return t.state
}

// Context is cancelled when the Tailor closes for any reason.
// CRC: crc-Tailor.md | R190
func (t *Tailor) Context() context.Context { return t.ctx }

// Ref returns the Tailor's session reference.
// CRC: crc-Tailor.md | R194
func (t *Tailor) Ref() SessionRef { return t.ref }

// info snapshots the Tailor for PendingTailors.
// CRC: crc-Tailor.md | R192
func (t *Tailor) info() TailorInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	return TailorInfo{
		Name:       t.name,
		Created:    t.created,
		LastStitch: t.lastStitch,
		LastSite:   t.lastSite,
		LongLived:  t.longLived,
	}
}
