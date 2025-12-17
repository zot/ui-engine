// CRC: crc-MessageRelay.md
// Spec: protocol.md, deployment.md
package server

import (
	"sync"

	"github.com/zot/ui/internal/protocol"
	"github.com/zot/ui/internal/variable"
)

// MessageRelay forwards messages between frontend and backend connections.
// NOTE: This is placeholder code for future proxied backend support.
// Currently not used - watch functionality is now per-session via Backend interface.
type MessageRelay struct {
	store   *variable.Store
	pending *PendingQueueManager

	// Track which variables are bound to backends
	boundVariables map[int64]string // varID -> backendConnID
	mu             sync.RWMutex
}

// NewMessageRelay creates a new message relay.
func NewMessageRelay(store *variable.Store, pending *PendingQueueManager) *MessageRelay {
	return &MessageRelay{
		store:          store,
		pending:        pending,
		boundVariables: make(map[int64]string),
	}
}

// RegisterBoundVariable marks a variable as bound to a backend connection.
func (r *MessageRelay) RegisterBoundVariable(varID int64, backendConnID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.boundVariables[varID] = backendConnID
}

// UnregisterBoundVariable removes a variable from bound tracking.
func (r *MessageRelay) UnregisterBoundVariable(varID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.boundVariables, varID)
}

// UnregisterBackend removes all variables bound to a backend connection.
func (r *MessageRelay) UnregisterBackend(backendConnID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for varID, connID := range r.boundVariables {
		if connID == backendConnID {
			delete(r.boundVariables, varID)
		}
	}
}

// GetVariableHolder returns the backend connection holding a variable, or empty if unbound.
func (r *MessageRelay) GetVariableHolder(varID int64) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.boundVariables[varID]
}

// IsBound checks if a variable is bound to a backend.
func (r *MessageRelay) IsBound(varID int64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.boundVariables[varID]
	return ok
}

// ForwardingDecision represents how a message should be forwarded.
type ForwardingDecision struct {
	ShouldForward  bool   // Whether to forward the message
	TargetConnID   string // Specific backend to forward to (empty = all backends)
	HandleLocally  bool   // Whether to also handle locally
}

// ShouldForward determines if a message should be forwarded and to where.
// Based on user-clarified rules:
// - watch/unwatch: Only to variable holder (backend for bound, local for unbound)
// - create/destroy/update/get/getObjects: Forward regardless of where variables are held
func (r *MessageRelay) ShouldForward(msgType protocol.MessageType, varID int64) ForwardingDecision {
	holder := r.GetVariableHolder(varID)
	isBound := holder != ""

	switch msgType {
	case protocol.MsgWatch, protocol.MsgUnwatch:
		// Watch/unwatch only go to variable holder
		if isBound {
			return ForwardingDecision{
				ShouldForward: true,
				TargetConnID:  holder,
				HandleLocally: true, // Also track locally for watch tallying
			}
		}
		// Unbound: handle locally only
		return ForwardingDecision{
			ShouldForward: false,
			HandleLocally: true,
		}

	case protocol.MsgCreate, protocol.MsgDestroy, protocol.MsgUpdate, protocol.MsgGet, protocol.MsgGetObjects:
		// These are always forwarded regardless of where variables are held
		return ForwardingDecision{
			ShouldForward: true,
			TargetConnID:  "", // Forward to all backends or handle request
			HandleLocally: true,
		}

	default:
		// Unknown messages are handled locally only
		return ForwardingDecision{
			ShouldForward: false,
			HandleLocally: true,
		}
	}
}

// RelayToBackend determines if a frontend message should be relayed to a backend.
func (r *MessageRelay) RelayToBackend(msg *protocol.Message, varID int64) *ForwardingDecision {
	decision := r.ShouldForward(msg.Type, varID)
	if !decision.ShouldForward {
		return nil
	}
	return &decision
}

// EnqueuePending adds a push message to the pending queue for a connection.
func (r *MessageRelay) EnqueuePending(connectionID string, msg *protocol.Message) {
	if r.pending != nil {
		r.pending.Enqueue(connectionID, msg)
	}
}

// EnqueuePendingToWatchers enqueues a message to all watchers of a variable.
// NOTE: This needs backend-specific implementation for proxied backends.
// For hosted (Lua) backends, use session.GetBackend().GetWatchers() instead.
func (r *MessageRelay) EnqueuePendingToWatchers(varID int64, msg *protocol.Message, excludeConnID string, watchers []string) {
	for _, connID := range watchers {
		if connID != excludeConnID {
			r.EnqueuePending(connID, msg)
		}
	}
}
