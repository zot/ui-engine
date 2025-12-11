// CRC: crc-MessageBatcher.md
// Spec: protocol.md
package protocol

import (
	"encoding/json"
	"strings"
	"sync"
)

// Priority levels for batching
type Priority int

const (
	PriorityHigh   Priority = 0
	PriorityMedium Priority = 1
	PriorityLow    Priority = 2
)

// ParsePrioritySuffix extracts priority from property name suffix.
// Returns base property name and priority.
// Examples: "viewdefs:high" -> ("viewdefs", PriorityHigh)
//
//	"name" -> ("name", PriorityMedium)
func ParsePrioritySuffix(propertyName string) (string, Priority) {
	if strings.HasSuffix(propertyName, ":high") {
		return strings.TrimSuffix(propertyName, ":high"), PriorityHigh
	}
	if strings.HasSuffix(propertyName, ":med") {
		return strings.TrimSuffix(propertyName, ":med"), PriorityMedium
	}
	if strings.HasSuffix(propertyName, ":low") {
		return strings.TrimSuffix(propertyName, ":low"), PriorityLow
	}
	return propertyName, PriorityMedium
}

// PendingChange represents a queued change for a variable.
type PendingChange struct {
	VarID         int64
	Value         json.RawMessage
	ValuePriority Priority
	HasValue      bool
	Properties    map[string]string     // property name -> value
	PropPriorities map[string]Priority  // property name -> priority
}

// MessageBatcher batches protocol messages by priority.
type MessageBatcher struct {
	pending map[int64]*PendingChange
	mu      sync.Mutex
}

// NewMessageBatcher creates a new message batcher.
func NewMessageBatcher() *MessageBatcher {
	return &MessageBatcher{
		pending: make(map[int64]*PendingChange),
	}
}

// getOrCreate returns existing pending change or creates a new one.
func (b *MessageBatcher) getOrCreate(varID int64) *PendingChange {
	if pc, ok := b.pending[varID]; ok {
		return pc
	}
	pc := &PendingChange{
		VarID:          varID,
		ValuePriority:  PriorityMedium,
		Properties:     make(map[string]string),
		PropPriorities: make(map[string]Priority),
	}
	b.pending[varID] = pc
	return pc
}

// QueueValue queues a value change with the given priority.
func (b *MessageBatcher) QueueValue(varID int64, value json.RawMessage, priority Priority) {
	b.mu.Lock()
	defer b.mu.Unlock()

	pc := b.getOrCreate(varID)
	pc.Value = value
	pc.ValuePriority = priority
	pc.HasValue = true
}

// QueueProperty queues a property change.
// Property name can include priority suffix (e.g., "viewdefs:high").
func (b *MessageBatcher) QueueProperty(varID int64, propertyName, value string) {
	baseName, priority := ParsePrioritySuffix(propertyName)

	b.mu.Lock()
	defer b.mu.Unlock()

	pc := b.getOrCreate(varID)
	pc.Properties[baseName] = value
	pc.PropPriorities[baseName] = priority
}

// QueueProperties queues multiple property changes.
func (b *MessageBatcher) QueueProperties(varID int64, properties map[string]string) {
	for name, value := range properties {
		b.QueueProperty(varID, name, value)
	}
}

// IsEmpty returns true if no changes are pending.
func (b *MessageBatcher) IsEmpty() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending) == 0
}

// batchEntry represents a single update in the batch.
type batchEntry struct {
	priority Priority
	message  *Message
}

// Flush builds and returns the batched messages, clearing pending state.
// Returns nil if no changes are pending.
func (b *MessageBatcher) Flush() []*Message {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.pending) == 0 {
		return nil
	}

	// Collect all entries
	var entries []batchEntry

	for _, pc := range b.pending {
		// Group properties by priority
		highProps := make(map[string]string)
		medProps := make(map[string]string)
		lowProps := make(map[string]string)

		for name, value := range pc.Properties {
			priority := pc.PropPriorities[name]
			switch priority {
			case PriorityHigh:
				highProps[name] = value
			case PriorityMedium:
				medProps[name] = value
			case PriorityLow:
				lowProps[name] = value
			}
		}

		// Create messages for each priority level that has content
		// High priority
		if len(highProps) > 0 || (pc.HasValue && pc.ValuePriority == PriorityHigh) {
			msg := b.createUpdateMessage(pc.VarID,
				pc.HasValue && pc.ValuePriority == PriorityHigh,
				pc.Value,
				highProps)
			if msg != nil {
				entries = append(entries, batchEntry{PriorityHigh, msg})
			}
		}

		// Medium priority
		if len(medProps) > 0 || (pc.HasValue && pc.ValuePriority == PriorityMedium) {
			msg := b.createUpdateMessage(pc.VarID,
				pc.HasValue && pc.ValuePriority == PriorityMedium,
				pc.Value,
				medProps)
			if msg != nil {
				entries = append(entries, batchEntry{PriorityMedium, msg})
			}
		}

		// Low priority
		if len(lowProps) > 0 || (pc.HasValue && pc.ValuePriority == PriorityLow) {
			msg := b.createUpdateMessage(pc.VarID,
				pc.HasValue && pc.ValuePriority == PriorityLow,
				pc.Value,
				lowProps)
			if msg != nil {
				entries = append(entries, batchEntry{PriorityLow, msg})
			}
		}
	}

	// Clear pending
	b.pending = make(map[int64]*PendingChange)

	// Sort by priority (stable sort to preserve order within priority)
	// Simple insertion sort since we have only 3 priority levels
	result := make([]*Message, 0, len(entries))
	for p := PriorityHigh; p <= PriorityLow; p++ {
		for _, e := range entries {
			if e.priority == p {
				result = append(result, e.message)
			}
		}
	}

	return result
}

// createUpdateMessage creates an update message for a variable.
func (b *MessageBatcher) createUpdateMessage(varID int64, includeValue bool, value json.RawMessage, properties map[string]string) *Message {
	// Skip if nothing to send
	if !includeValue && len(properties) == 0 {
		return nil
	}

	update := UpdateMessage{
		VarID: varID,
	}
	if includeValue {
		update.Value = value
	}
	if len(properties) > 0 {
		update.Properties = properties
	}

	msg, _ := NewMessage(MsgUpdate, update)
	return msg
}

// FlushJSON returns the batch as a JSON array or single message.
// Returns nil if no changes are pending.
func (b *MessageBatcher) FlushJSON() ([]byte, error) {
	messages := b.Flush()
	if messages == nil || len(messages) == 0 {
		return nil, nil
	}

	if len(messages) == 1 {
		return messages[0].Encode()
	}

	return json.Marshal(messages)
}
