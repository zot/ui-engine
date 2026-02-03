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

// MessageSender sends protocol messages and logs.
type MessageSender interface {
	Send(connectionID string, msg *protocol.Message) error
	SendBatch(connectionID string, msgs []*protocol.Message) error
	Log(level int, format string, args ...interface{})
}

// pendingUpdate holds an update message and its target watchers.
type pendingUpdate struct {
	msg      *protocol.Message
	watchers []string // connection IDs to send to
}

// OutgoingBatcher batches outgoing messages for a single session with debouncing.
// User events trigger immediate flush; non-user events are debounced.
// Each session has its own batcher instance.
type OutgoingBatcher struct {
	mu               sync.Mutex
	pendingUpdates   []pendingUpdate // pending updates with watchers
	debounceTimer    *time.Timer     // debounce timer
	debounceInterval time.Duration
	sender           MessageSender // collaborator for sending messages
	batchCount       int
}

// NewOutgoingBatcher creates a batcher with the given message sender.
func NewOutgoingBatcher(sender MessageSender) *OutgoingBatcher {
	return &OutgoingBatcher{
		debounceInterval: 10 * time.Millisecond,
		sender:           sender,
	}
}

// Queue adds a message to the pending queue and starts debounce timer.
// watchers is the list of connection IDs to send this message to.
func (b *OutgoingBatcher) Queue(msg *protocol.Message, watchers []string) {
	if msg == nil || len(watchers) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Add to pending queue
	b.pendingUpdates = append(b.pendingUpdates, pendingUpdate{
		msg:      msg,
		watchers: watchers,
	})

	// Only start timer if not already running (may have been pre-started)
	if b.debounceTimer == nil {
		b.debounceTimer = time.AfterFunc(b.debounceInterval, func() {
			b.flush()
		})
	}
}

// EnsureDebounceStarted ensures the debounce timer is running.
// Called before processing to run debounce concurrently with processing.
// If timer already running, does nothing (preserves existing deadline).
func (b *OutgoingBatcher) EnsureDebounceStarted() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Only start timer if not already running
	if b.debounceTimer == nil {
		b.debounceTimer = time.AfterFunc(b.debounceInterval, func() {
			b.flush()
		})
	}
}

// FlushNow immediately sends all pending messages.
func (b *OutgoingBatcher) FlushNow() {
	b.mu.Lock()
	// Cancel debounce timer if running
	if b.debounceTimer != nil {
		b.debounceTimer.Stop()
	}
	b.mu.Unlock()

	b.flush()
}

// flush sends pending messages (called by timer or FlushNow).
// Groups messages by connection and sends one batch per connection.
func (b *OutgoingBatcher) flush() {
	b.mu.Lock()

	// Clear timer reference
	b.debounceTimer = nil

	// Get and clear pending updates
	updates := b.pendingUpdates
	b.pendingUpdates = nil

	b.batchCount += 1
	count := b.batchCount
	b.mu.Unlock()

	if len(updates) == 0 {
		return
	}

	// Group messages by connection ID
	connMsgs := make(map[string][]*protocol.Message)
	for _, update := range updates {
		for _, connID := range update.watchers {
			connMsgs[connID] = append(connMsgs[connID], update.msg)
		}
	}

	b.sender.Log(4, "[OUT] BATCH %d", count)
	// Send one batch per connection
	for connID, msgs := range connMsgs {
		if len(msgs) == 1 {
			b.sender.Send(connID, msgs[0])
		} else {
			b.sender.SendBatch(connID, msgs)
		}
	}
}

// Clear removes all pending messages and stops the timer.
// Called when session is destroyed.
func (b *OutgoingBatcher) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.debounceTimer != nil {
		b.debounceTimer.Stop()
	}
	b.debounceTimer = nil
	b.pendingUpdates = nil
}

// PendingCount returns the number of pending updates (for testing).
func (b *OutgoingBatcher) PendingCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pendingUpdates)
}
