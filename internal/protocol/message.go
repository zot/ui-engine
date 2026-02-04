// Package protocol implements the Variable Protocol message handling.
// CRC: crc-ProtocolHandler.md
// Spec: protocol.md
package protocol

import (
	"encoding/json"
)

// MessageType identifies the type of protocol message.
type MessageType string

const (
	// Relayed messages (frontend <-> UI server <-> backend)
	MsgCreate  MessageType = "create"
	MsgDestroy MessageType = "destroy"
	MsgUpdate  MessageType = "update"
	MsgWatch   MessageType = "watch"
	MsgUnwatch MessageType = "unwatch"

	// Server-response messages
	MsgError MessageType = "error"

	// UI server-handled messages (not relayed)
	MsgGet        MessageType = "get"
	MsgGetObjects MessageType = "getObjects"
	MsgPoll       MessageType = "poll"
)

// Message is the base protocol message structure.
type Message struct {
	Type MessageType     `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// CreateMessage represents a create variable request.
// Spec: protocol.md - create(id, parentId, value, properties, nowatch?, unbound?)
type CreateMessage struct {
	ID         int64             `json:"id"`
	ParentID   int64             `json:"parentId,omitempty"`
	Value      json.RawMessage   `json:"value,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	NoWatch    bool              `json:"nowatch,omitempty"`
	Unbound    bool              `json:"unbound,omitempty"`
}

// DestroyMessage represents a destroy variable request.
type DestroyMessage struct {
	VarID int64 `json:"varId"`
}

// UpdateMessage represents an update variable request.
type UpdateMessage struct {
	VarID      int64             `json:"varId"`
	Value      json.RawMessage   `json:"value,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// WatchMessage represents a watch/unwatch request.
type WatchMessage struct {
	VarID int64 `json:"varId"`
}

// GetMessage represents a get variables request.
type GetMessage struct {
	VarIDs []int64 `json:"varIds"`
}

// GetResponse contains variable values.
type GetResponse struct {
	Variables []VariableData `json:"variables"`
}

// VariableData contains a variable's data for get responses.
type VariableData struct {
	ID         int64             `json:"id"`
	Value      json.RawMessage   `json:"value,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// GetObjectsMessage represents a get objects by ID request.
type GetObjectsMessage struct {
	ObjIDs []int64 `json:"objIds"`
}

// GetObjectsResponse contains object data.
type GetObjectsResponse struct {
	Objects []ObjectData `json:"objects"`
}

// ObjectData contains an object's data.
type ObjectData struct {
	ID    int64           `json:"obj"`
	Value json.RawMessage `json:"value"`
}

// PollMessage represents a poll for pending responses request.
type PollMessage struct {
	Wait string `json:"wait,omitempty"` // Duration string for long-polling
}

// ErrorMessage represents an error response.
// Spec: protocol.md - error(varId, code, description)
type ErrorMessage struct {
	VarID       int64  `json:"varId,omitempty"`
	Code        string `json:"code"`        // One-word error code (e.g., "path-failure", "not-found", "unauthorized")
	Description string `json:"description"` // Human-readable error description
}

// Response wraps handler responses (primarily for error reporting).
type Response struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// BatchWrapper wraps a batch of messages with a userEvent flag.
// Spec: protocol.md - Frontend sends batches with userEvent flag for immediate/debounced response
type BatchWrapper struct {
	UserEvent bool      `json:"userEvent"`
	Messages  []Message `json:"messages"`
}

// ParseMessage parses a raw JSON message into a typed message.
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// ParseMessages parses raw JSON that may be a single message, batched array, or batch wrapper.
// Returns messages and userEvent flag (true if user-triggered, false otherwise).
// Spec: protocol.md - Messages can be sent individually or batched with userEvent flag
func ParseMessages(data []byte) ([]*Message, bool, error) {
	if len(data) == 0 {
		return nil, false, nil
	}

	// Check first byte to determine format
	switch data[0] {
	case '[':
		// Legacy: array format (treat as non-user event)
		var msgs []Message
		if err := json.Unmarshal(data, &msgs); err != nil {
			return nil, false, err
		}
		result := make([]*Message, len(msgs))
		for i := range msgs {
			result[i] = &msgs[i]
		}
		return result, false, nil

	case '{':
		// Could be wrapper or single message - try wrapper first
		var wrapper BatchWrapper
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return nil, false, err
		}

		// Check if it has messages field (wrapper format)
		if len(wrapper.Messages) > 0 {
			result := make([]*Message, len(wrapper.Messages))
			for i := range wrapper.Messages {
				result[i] = &wrapper.Messages[i]
			}
			return result, wrapper.UserEvent, nil
		}

		// Single message (legacy - treat as non-user event)
		msg, err := ParseMessage(data)
		if err != nil {
			return nil, false, err
		}
		return []*Message{msg}, false, nil

	default:
		return nil, false, nil
	}
}

// NewMessage creates a new message with the given type and data.
func NewMessage(msgType MessageType, data interface{}) (*Message, error) {
	var raw json.RawMessage
	if data != nil {
		var err error
		raw, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}
	return &Message{
		Type: msgType,
		Data: raw,
	}, nil
}

// Encode serializes a message to JSON.
func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}
