// CRC: crc-PendingResponseQueue.md
// Spec: deployment.md
package server

import (
	"sync"
	"time"

	"github.com/zot/ui-engine/internal/protocol"
)

// PendingResponseQueue accumulates push messages for polling clients.
type PendingResponseQueue struct {
	queue   []*protocol.Message
	waiters []chan struct{}
	mu      sync.Mutex
}

// NewPendingResponseQueue creates a new pending response queue.
func NewPendingResponseQueue() *PendingResponseQueue {
	return &PendingResponseQueue{
		queue:   make([]*protocol.Message, 0),
		waiters: make([]chan struct{}, 0),
	}
}

// Enqueue adds a message to the pending queue.
// Valid message types: update, error, destroy
func (q *PendingResponseQueue) Enqueue(msg *protocol.Message) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.queue = append(q.queue, msg)

	// Notify any waiters
	for _, ch := range q.waiters {
		select {
		case ch <- struct{}{}:
		default:
			// Waiter already notified or channel full
		}
	}
}

// Drain returns all pending messages and clears the queue.
func (q *PendingResponseQueue) Drain() []*protocol.Message {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) == 0 {
		return nil
	}

	messages := q.queue
	q.queue = make([]*protocol.Message, 0)
	return messages
}

// Poll returns pending messages, optionally waiting for availability.
// If wait is 0, returns immediately. Otherwise waits up to the duration.
func (q *PendingResponseQueue) Poll(wait time.Duration) []*protocol.Message {
	// Try immediate drain first
	messages := q.Drain()
	if len(messages) > 0 || wait == 0 {
		return messages
	}

	// Set up waiter channel
	ch := make(chan struct{}, 1)
	q.mu.Lock()
	q.waiters = append(q.waiters, ch)
	q.mu.Unlock()

	// Wait for notification or timeout
	select {
	case <-ch:
		// Message arrived
	case <-time.After(wait):
		// Timeout
	}

	// Remove waiter
	q.mu.Lock()
	for i, w := range q.waiters {
		if w == ch {
			q.waiters = append(q.waiters[:i], q.waiters[i+1:]...)
			break
		}
	}
	q.mu.Unlock()

	// Drain whatever is available
	return q.Drain()
}

// IsEmpty checks if the queue has pending messages.
func (q *PendingResponseQueue) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queue) == 0
}

// Len returns the number of pending messages.
func (q *PendingResponseQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queue)
}

// PendingQueueManager manages pending queues per connection.
type PendingQueueManager struct {
	queues map[string]*PendingResponseQueue
	mu     sync.RWMutex
}

// NewPendingQueueManager creates a new pending queue manager.
func NewPendingQueueManager() *PendingQueueManager {
	return &PendingQueueManager{
		queues: make(map[string]*PendingResponseQueue),
	}
}

// GetQueue returns the queue for a connection, creating if needed.
func (m *PendingQueueManager) GetQueue(connectionID string) *PendingResponseQueue {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, ok := m.queues[connectionID]
	if !ok {
		q = NewPendingResponseQueue()
		m.queues[connectionID] = q
	}
	return q
}

// RemoveQueue removes a connection's queue.
func (m *PendingQueueManager) RemoveQueue(connectionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.queues, connectionID)
}

// EnqueueToAll enqueues a message to all queues.
func (m *PendingQueueManager) EnqueueToAll(msg *protocol.Message) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, q := range m.queues {
		q.Enqueue(msg)
	}
}

// EnqueueTo enqueues a message to specific connections.
func (m *PendingQueueManager) EnqueueTo(msg *protocol.Message, connectionIDs []string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, connID := range connectionIDs {
		if q, ok := m.queues[connID]; ok {
			q.Enqueue(msg)
		}
	}
}

// Enqueue implements protocol.PendingQueuer interface.
func (m *PendingQueueManager) Enqueue(connectionID string, msg *protocol.Message) {
	q := m.GetQueue(connectionID)
	q.Enqueue(msg)
}

// Poll implements protocol.PendingQueuer interface.
func (m *PendingQueueManager) Poll(connectionID string, wait time.Duration) []*protocol.Message {
	q := m.GetQueue(connectionID)
	return q.Poll(wait)
}
