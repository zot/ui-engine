// Test Design: crc-ServerOutgoingBatcher.md
// Spec: protocol.md
// Sequence: seq-frontend-outgoing-batch.md
package server

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/zot/ui-engine/internal/protocol"
)

// mockSender implements MessageSender for testing
type mockSender struct {
	mu       sync.Mutex
	messages []*protocol.Message
	connIDs  []string
	onSend   func(connID string, msg *protocol.Message)
}

func (m *mockSender) Send(connectionID string, msg *protocol.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	m.connIDs = append(m.connIDs, connectionID)
	if m.onSend != nil {
		m.onSend(connectionID, msg)
	}
	return nil
}

func (m *mockSender) SendBatch(connectionID string, msgs []*protocol.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, msg := range msgs {
		m.messages = append(m.messages, msg)
		m.connIDs = append(m.connIDs, connectionID)
	}
	return nil
}

func (m *mockSender) Log(level int, format string, args ...interface{}) {
	// No-op for tests
}

func (m *mockSender) messageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func (m *mockSender) connCount(connID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, id := range m.connIDs {
		if id == connID {
			count++
		}
	}
	return count
}

// TestOutgoingBatcherQueue verifies basic queuing
func TestOutgoingBatcherQueue(t *testing.T) {
	mock := &mockSender{}
	batcher := NewOutgoingBatcher(mock)

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
		Value: json.RawMessage(`"test"`),
	})

	batcher.Queue(msg, []string{"conn1"})

	if batcher.PendingCount() != 1 {
		t.Errorf("Expected 1 pending, got %d", batcher.PendingCount())
	}

	// Wait for debounce timer
	time.Sleep(20 * time.Millisecond)

	if mock.messageCount() != 1 {
		t.Errorf("Expected 1 sent message, got %d", mock.messageCount())
	}

	if batcher.PendingCount() != 0 {
		t.Error("Should have no pending after flush")
	}
}

// TestOutgoingBatcherFlushNow verifies immediate flush
func TestOutgoingBatcherFlushNow(t *testing.T) {
	mock := &mockSender{}
	batcher := NewOutgoingBatcher(mock)

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
		Value: json.RawMessage(`"test"`),
	})

	batcher.Queue(msg, []string{"conn1"})
	batcher.FlushNow()

	if mock.messageCount() != 1 {
		t.Errorf("Expected 1 sent message after FlushNow, got %d", mock.messageCount())
	}

	if batcher.PendingCount() != 0 {
		t.Error("Should have no pending after FlushNow")
	}
}

// TestOutgoingBatcherDebounce verifies debounce behavior
func TestOutgoingBatcherDebounce(t *testing.T) {
	mock := &mockSender{}
	batcher := NewOutgoingBatcher(mock)

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
	})

	// Queue multiple messages quickly
	batcher.Queue(msg, []string{"conn1"})
	batcher.Queue(msg, []string{"conn1"})
	batcher.Queue(msg, []string{"conn1"})

	// Should still be pending (debounce not fired yet)
	if batcher.PendingCount() != 3 {
		t.Errorf("Expected 3 pending, got %d", batcher.PendingCount())
	}

	if mock.messageCount() != 0 {
		t.Error("Should not have sent yet (debounce)")
	}

	// Wait for debounce
	time.Sleep(20 * time.Millisecond)

	if mock.messageCount() != 3 {
		t.Errorf("Expected 3 messages sent, got %d", mock.messageCount())
	}
}

// TestOutgoingBatcherMultipleWatchers verifies messages sent to all watchers
func TestOutgoingBatcherMultipleWatchers(t *testing.T) {
	mock := &mockSender{}
	batcher := NewOutgoingBatcher(mock)

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
	})

	// Queue with multiple watchers
	batcher.Queue(msg, []string{"conn1", "conn2", "conn3"})
	batcher.FlushNow()

	if mock.connCount("conn1") != 1 {
		t.Errorf("conn1 should receive 1, got %d", mock.connCount("conn1"))
	}
	if mock.connCount("conn2") != 1 {
		t.Errorf("conn2 should receive 1, got %d", mock.connCount("conn2"))
	}
	if mock.connCount("conn3") != 1 {
		t.Errorf("conn3 should receive 1, got %d", mock.connCount("conn3"))
	}
}

// TestOutgoingBatcherClear verifies cleanup
func TestOutgoingBatcherClear(t *testing.T) {
	mock := &mockSender{}
	batcher := NewOutgoingBatcher(mock)

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
	})

	batcher.Queue(msg, []string{"conn1"})
	batcher.Clear()

	if batcher.PendingCount() != 0 {
		t.Error("Should have no pending after Clear")
	}

	// Wait longer than debounce to ensure timer was cancelled
	time.Sleep(20 * time.Millisecond)

	if mock.messageCount() != 0 {
		t.Error("Should not have sent after Clear")
	}
}

func TestEnsureDebounceStarted(t *testing.T) {
	mock := &mockSender{}
	batcher := NewOutgoingBatcher(mock)

	// EnsureDebounceStarted should start timer
	batcher.EnsureDebounceStarted()

	// Calling again should NOT restart timer (same deadline preserved)
	batcher.EnsureDebounceStarted()
	batcher.EnsureDebounceStarted()

	// Timer is running but no messages queued
	if batcher.PendingCount() != 0 {
		t.Errorf("Expected 0 pending, got %d", batcher.PendingCount())
	}

	// Wait for debounce to fire
	time.Sleep(20 * time.Millisecond)

	// Should have flushed (but nothing to send)
	if mock.messageCount() != 0 {
		t.Errorf("Expected 0 messages sent (nothing queued), got %d", mock.messageCount())
	}
}

func TestQueuePreservesPreStartedTimer(t *testing.T) {
	var flushTime time.Time
	var mu sync.Mutex

	mock := &mockSender{
		onSend: func(connID string, msg *protocol.Message) {
			mu.Lock()
			if flushTime.IsZero() {
				flushTime = time.Now()
			}
			mu.Unlock()
		},
	}
	batcher := NewOutgoingBatcher(mock)

	// Pre-start the debounce timer
	startTime := time.Now()
	batcher.EnsureDebounceStarted()

	// Simulate some processing time
	time.Sleep(5 * time.Millisecond)

	// Queue a message - should NOT restart the timer
	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
	})
	batcher.Queue(msg, []string{"conn1"})

	// Wait for flush
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	elapsed := flushTime.Sub(startTime)
	mu.Unlock()

	// Timer should have fired ~10ms after startTime, not ~15ms (5ms processing + 10ms new timer)
	// Allow some slack for test timing
	if elapsed > 15*time.Millisecond {
		t.Errorf("Timer appears to have been restarted: flush took %v from start", elapsed)
	}
}

// TestPerSessionBatcherIsolation verifies that each session has its own batcher
func TestPerSessionBatcherIsolation(t *testing.T) {
	mock1 := &mockSender{}
	mock2 := &mockSender{}

	// Each session gets its own batcher
	batcher1 := NewOutgoingBatcher(mock1)
	batcher2 := NewOutgoingBatcher(mock2)

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
	})

	// Queue to different batchers
	batcher1.Queue(msg, []string{"conn1"})
	batcher2.Queue(msg, []string{"conn2"})

	// Flush only batcher1
	batcher1.FlushNow()

	if mock1.connCount("conn1") != 1 {
		t.Errorf("batcher1 should have sent to conn1, got %d", mock1.connCount("conn1"))
	}
	if mock2.connCount("conn2") != 0 {
		t.Errorf("batcher2 should not have sent yet, got %d", mock2.connCount("conn2"))
	}

	// Wait for batcher2 debounce
	time.Sleep(20 * time.Millisecond)

	if mock2.connCount("conn2") != 1 {
		t.Errorf("batcher2 should have sent after debounce, got %d", mock2.connCount("conn2"))
	}
}
