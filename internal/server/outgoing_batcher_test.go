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

// TestServerOutgoingBatcherQueue verifies basic queuing
func TestServerOutgoingBatcherQueue(t *testing.T) {
	var sent []*protocol.Message
	var mu sync.Mutex

	batcher := NewServerOutgoingBatcher(func(connID string, msg *protocol.Message) {
		mu.Lock()
		sent = append(sent, msg)
		mu.Unlock()
	})

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
		Value: json.RawMessage(`"test"`),
	})

	batcher.Queue("session1", msg, []string{"conn1"})

	if batcher.PendingCount("session1") != 1 {
		t.Errorf("Expected 1 pending, got %d", batcher.PendingCount("session1"))
	}

	// Wait for debounce timer
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	if len(sent) != 1 {
		t.Errorf("Expected 1 sent message, got %d", len(sent))
	}
	mu.Unlock()

	if batcher.PendingCount("session1") != 0 {
		t.Error("Should have no pending after flush")
	}
}

// TestServerOutgoingBatcherFlushNow verifies immediate flush
func TestServerOutgoingBatcherFlushNow(t *testing.T) {
	var sent []*protocol.Message
	var mu sync.Mutex

	batcher := NewServerOutgoingBatcher(func(connID string, msg *protocol.Message) {
		mu.Lock()
		sent = append(sent, msg)
		mu.Unlock()
	})

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
		Value: json.RawMessage(`"test"`),
	})

	batcher.Queue("session1", msg, []string{"conn1"})
	batcher.FlushNow("session1")

	mu.Lock()
	if len(sent) != 1 {
		t.Errorf("Expected 1 sent message after FlushNow, got %d", len(sent))
	}
	mu.Unlock()

	if batcher.PendingCount("session1") != 0 {
		t.Error("Should have no pending after FlushNow")
	}
}

// TestServerOutgoingBatcherDebounce verifies debounce behavior
func TestServerOutgoingBatcherDebounce(t *testing.T) {
	var sendCount int
	var mu sync.Mutex

	batcher := NewServerOutgoingBatcher(func(connID string, msg *protocol.Message) {
		mu.Lock()
		sendCount++
		mu.Unlock()
	})

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
	})

	// Queue multiple messages quickly
	batcher.Queue("session1", msg, []string{"conn1"})
	batcher.Queue("session1", msg, []string{"conn1"})
	batcher.Queue("session1", msg, []string{"conn1"})

	// Should still be pending (debounce not fired yet)
	if batcher.PendingCount("session1") != 3 {
		t.Errorf("Expected 3 pending, got %d", batcher.PendingCount("session1"))
	}

	mu.Lock()
	if sendCount != 0 {
		t.Error("Should not have sent yet (debounce)")
	}
	mu.Unlock()

	// Wait for debounce
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	if sendCount != 3 {
		t.Errorf("Expected 3 messages sent, got %d", sendCount)
	}
	mu.Unlock()
}

// TestServerOutgoingBatcherMultipleWatchers verifies messages sent to all watchers
func TestServerOutgoingBatcherMultipleWatchers(t *testing.T) {
	receivedBy := make(map[string]int)
	var mu sync.Mutex

	batcher := NewServerOutgoingBatcher(func(connID string, msg *protocol.Message) {
		mu.Lock()
		receivedBy[connID]++
		mu.Unlock()
	})

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
	})

	// Queue with multiple watchers
	batcher.Queue("session1", msg, []string{"conn1", "conn2", "conn3"})
	batcher.FlushNow("session1")

	mu.Lock()
	if receivedBy["conn1"] != 1 {
		t.Errorf("conn1 should receive 1, got %d", receivedBy["conn1"])
	}
	if receivedBy["conn2"] != 1 {
		t.Errorf("conn2 should receive 1, got %d", receivedBy["conn2"])
	}
	if receivedBy["conn3"] != 1 {
		t.Errorf("conn3 should receive 1, got %d", receivedBy["conn3"])
	}
	mu.Unlock()
}

// TestServerOutgoingBatcherClearSession verifies session cleanup
func TestServerOutgoingBatcherClearSession(t *testing.T) {
	sendCount := 0
	batcher := NewServerOutgoingBatcher(func(connID string, msg *protocol.Message) {
		sendCount++
	})

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
	})

	batcher.Queue("session1", msg, []string{"conn1"})
	batcher.ClearSession("session1")

	if batcher.PendingCount("session1") != 0 {
		t.Error("Should have no pending after ClearSession")
	}

	// Wait longer than debounce to ensure timer was cancelled
	time.Sleep(20 * time.Millisecond)

	if sendCount != 0 {
		t.Error("Should not have sent after ClearSession")
	}
}

// TestServerOutgoingBatcherPerSessionIsolation verifies sessions are independent
func TestServerOutgoingBatcherPerSessionIsolation(t *testing.T) {
	session1Count := 0
	session2Count := 0
	var mu sync.Mutex

	batcher := NewServerOutgoingBatcher(func(connID string, msg *protocol.Message) {
		mu.Lock()
		if connID == "conn1" {
			session1Count++
		} else if connID == "conn2" {
			session2Count++
		}
		mu.Unlock()
	})

	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
	})

	// Queue to different sessions
	batcher.Queue("session1", msg, []string{"conn1"})
	batcher.Queue("session2", msg, []string{"conn2"})

	// Flush only session1
	batcher.FlushNow("session1")

	mu.Lock()
	if session1Count != 1 {
		t.Errorf("session1 should have 1 sent, got %d", session1Count)
	}
	if session2Count != 0 {
		t.Errorf("session2 should have 0 sent (not flushed yet), got %d", session2Count)
	}
	mu.Unlock()

	// Wait for session2 debounce
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	if session2Count != 1 {
		t.Errorf("session2 should have 1 sent after debounce, got %d", session2Count)
	}
	mu.Unlock()
}

func TestEnsureDebounceStarted(t *testing.T) {
	var sendCount int
	var mu sync.Mutex

	batcher := NewServerOutgoingBatcher(func(connID string, msg *protocol.Message) {
		mu.Lock()
		sendCount++
		mu.Unlock()
	})

	// EnsureDebounceStarted should start timer
	batcher.EnsureDebounceStarted("session1")

	// Calling again should NOT restart timer (same deadline preserved)
	batcher.EnsureDebounceStarted("session1")
	batcher.EnsureDebounceStarted("session1")

	// Timer is running but no messages queued
	if batcher.PendingCount("session1") != 0 {
		t.Errorf("Expected 0 pending, got %d", batcher.PendingCount("session1"))
	}

	// Wait for debounce to fire
	time.Sleep(20 * time.Millisecond)

	// Should have flushed (but nothing to send)
	mu.Lock()
	if sendCount != 0 {
		t.Errorf("Expected 0 messages sent (nothing queued), got %d", sendCount)
	}
	mu.Unlock()
}

func TestQueuePreservesPreStartedTimer(t *testing.T) {
	var flushTime time.Time
	var mu sync.Mutex

	batcher := NewServerOutgoingBatcher(func(connID string, msg *protocol.Message) {
		mu.Lock()
		if flushTime.IsZero() {
			flushTime = time.Now()
		}
		mu.Unlock()
	})

	// Pre-start the debounce timer
	startTime := time.Now()
	batcher.EnsureDebounceStarted("session1")

	// Simulate some processing time
	time.Sleep(5 * time.Millisecond)

	// Queue a message - should NOT restart the timer
	msg, _ := protocol.NewMessage(protocol.MsgUpdate, protocol.UpdateMessage{
		VarID: 1,
	})
	batcher.Queue("session1", msg, []string{"conn1"})

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
