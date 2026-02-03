// Package server implements the UI server communication layer.
// CRC: crc-ServerOutgoingBatcher.md
// Spec: protocol.md
// Sequence: seq-frontend-outgoing-batch.md
package server

import (
	"sync"
	"time"

	"github.com/zot/ui-engine/internal/protocol"
)

// pendingUpdate holds an update message and its target watchers.
type pendingUpdate struct {
	msg      *protocol.Message
	watchers []string // connection IDs to send to
}

// ServerOutgoingBatcher batches outgoing messages per session with debouncing.
// User events trigger immediate flush; non-user events are debounced.
type ServerOutgoingBatcher struct {
	mu               sync.Mutex
	pendingUpdates   map[string][]pendingUpdate // sessionID -> pending updates with watchers
	debounceTimers   map[string]*time.Timer     // sessionID -> debounce timer
	debounceInterval time.Duration
	sendFn           func(connectionID string, msg *protocol.Message)
}

// NewServerOutgoingBatcher creates a batcher with the given send function.
func NewServerOutgoingBatcher(sendFn func(connectionID string, msg *protocol.Message)) *ServerOutgoingBatcher {
	return &ServerOutgoingBatcher{
		pendingUpdates:   make(map[string][]pendingUpdate),
		debounceTimers:   make(map[string]*time.Timer),
		debounceInterval: 10 * time.Millisecond,
		sendFn:           sendFn,
	}
}

// Queue adds a message to the session's pending queue and starts debounce timer.
// watchers is the list of connection IDs to send this message to.
func (b *ServerOutgoingBatcher) Queue(sessionID string, msg *protocol.Message, watchers []string) {
	if msg == nil || len(watchers) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Add to pending queue
	b.pendingUpdates[sessionID] = append(b.pendingUpdates[sessionID], pendingUpdate{
		msg:      msg,
		watchers: watchers,
	})

	// Only start timer if not already running (may have been pre-started)
	if b.debounceTimers[sessionID] == nil {
		b.debounceTimers[sessionID] = time.AfterFunc(b.debounceInterval, func() {
			b.flushSession(sessionID)
		})
	}
}

// EnsureDebounceStarted ensures the debounce timer is running for a session.
// Called before processing to run debounce concurrently with processing.
// If timer already running, does nothing (preserves existing deadline).
func (b *ServerOutgoingBatcher) EnsureDebounceStarted(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Only start timer if not already running
	if b.debounceTimers[sessionID] == nil {
		b.debounceTimers[sessionID] = time.AfterFunc(b.debounceInterval, func() {
			b.flushSession(sessionID)
		})
	}
}

// FlushNow immediately sends all pending messages for a session.
func (b *ServerOutgoingBatcher) FlushNow(sessionID string) {
	b.mu.Lock()
	// Cancel debounce timer if running
	if timer := b.debounceTimers[sessionID]; timer != nil {
		timer.Stop()
	}
	b.mu.Unlock()

	b.flushSession(sessionID)
}

// flushSession sends pending messages for a session (called by timer).
func (b *ServerOutgoingBatcher) flushSession(sessionID string) {
	b.mu.Lock()

	// Clear timer reference
	delete(b.debounceTimers, sessionID)

	// Get and clear pending updates
	updates := b.pendingUpdates[sessionID]
	delete(b.pendingUpdates, sessionID)

	b.mu.Unlock()

	// Send outside lock
	for _, update := range updates {
		for _, connID := range update.watchers {
			b.sendFn(connID, update.msg)
		}
	}
}

// ClearSession removes all pending messages and timers for a session.
func (b *ServerOutgoingBatcher) ClearSession(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if timer := b.debounceTimers[sessionID]; timer != nil {
		timer.Stop()
	}
	delete(b.debounceTimers, sessionID)
	delete(b.pendingUpdates, sessionID)
}

// PendingCount returns the number of pending updates for a session (for testing).
func (b *ServerOutgoingBatcher) PendingCount(sessionID string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pendingUpdates[sessionID])
}
