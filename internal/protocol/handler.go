// CRC: crc-ProtocolHandler.md
// Spec: protocol.md
package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zot/ui-engine/internal/backend"
	"github.com/zot/ui-engine/internal/config"
)

// MessageSender is an interface for sending messages to a connection.
type MessageSender interface {
	Send(connectionID string, msg *Message) error
	Broadcast(sessionID string, msg *Message) error
}

// PendingQueuer is an interface for pending message queues.
type PendingQueuer interface {
	Enqueue(connectionID string, msg *Message)
	Poll(connectionID string, wait time.Duration) []*Message
}

// PathVariableHandler handles frontend-created path variables.
type PathVariableHandler interface {
	// HandleFrontendCreate handles a path-based variable create from frontend.
	// The id is provided by the frontend (frontend-vended IDs).
	// Returns the resolved value and properties.
	HandleFrontendCreate(sessionID string, id int64, parentID int64, properties map[string]string) error

	// HandleFrontendUpdate handles an update to a path-based variable from frontend.
	// Updates the backend object via the variable's path and returns error if any.
	HandleFrontendUpdate(sessionID string, varID int64, value json.RawMessage, properties map[string]string) error
}

// BackendLookup provides per-connection backend lookup.
// Used by the protocol handler to route watch operations to the correct session's backend.
type BackendLookup interface {
	// GetBackendForConnection returns the backend for a connection.
	// Returns nil if connection is not associated with a session.
	GetBackendForConnection(connectionID string) backend.Backend
}

// CRC: crc-ProtocolHandler.md | R112, R113
// MessageQueuer queues outgoing messages through the session's OutgoingBatcher.
type MessageQueuer interface {
	// Queue adds a message to the batcher's pending queue for the given watchers.
	Queue(msg *Message, watchers []string)
}

// Handler processes protocol messages.
type Handler struct {
	config              *config.Config
	backendLookup       BackendLookup
	sender              MessageSender
	queuer              MessageQueuer
	pending             PendingQueuer
	pathVariableHandler PathVariableHandler // For path-based frontend creates
}

// NewHandler creates a new protocol handler.
func NewHandler(cfg *config.Config, sender MessageSender) *Handler {
	return &Handler{
		config: cfg,
		sender: sender,
	}
}

// SetBackendLookup sets the backend lookup for per-session watch operations.
func (h *Handler) SetBackendLookup(lookup BackendLookup) {
	h.backendLookup = lookup
}

// SetQueuer sets the message queuer for batched outgoing messages.
// CRC: crc-ProtocolHandler.md | R113
func (h *Handler) SetQueuer(queuer MessageQueuer) {
	h.queuer = queuer
}

// SetPendingQueuer sets the pending queue manager.
func (h *Handler) SetPendingQueuer(pending PendingQueuer) {
	h.pending = pending
}

// SetPathVariableHandler sets the handler for path-based frontend creates.
func (h *Handler) SetPathVariableHandler(handler PathVariableHandler) {
	h.pathVariableHandler = handler
}

// Log logs a message via the config.
func (h *Handler) Log(level int, format string, args ...interface{}) {
	h.config.Log(level, format, args...)
}

// HandleMessage processes an incoming protocol message.
func (h *Handler) HandleMessage(connectionID string, msg *Message) (*Response, error) {
	// Log message (verbosity level 2: abbreviated, level 4: complete)
	msgType := strings.ToUpper(string(msg.Type))
	if h.config.Verbosity() >= 4 {
		h.Log(4, "[IN] %s: from=%s data=%s", msgType, connectionID, string(msg.Data))
	} else {
		h.Log(2, "[IN] %s: from=%s", msgType, connectionID)
	}

	switch msg.Type {
	case MsgCreate:
		return h.handleCreate(connectionID, msg.Data)
	case MsgDestroy:
		return h.handleDestroy(connectionID, msg.Data)
	case MsgUpdate:
		return h.handleUpdate(connectionID, msg.Data)
	case MsgWatch:
		return h.handleWatch(connectionID, msg.Data)
	case MsgUnwatch:
		return h.handleUnwatch(connectionID, msg.Data)
	default:
		return nil, fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleCreate processes a create message.
// Spec: protocol.md - create(id, parentId, value, properties, nowatch?, unbound?)
// Frontend provides the variable ID (frontend-vended IDs).
func (h *Handler) handleCreate(connectionID string, data json.RawMessage) (*Response, error) {
	var msg CreateMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		h.Log(0, "ERROR unmarshalling CreateMessage from %s", string(data))
		return nil, err
	}

	id := msg.ID
	if id == 0 {
		return &Response{Error: "create message must include id"}, nil
	}

	if h.pathVariableHandler != nil {
		// Path-based variable: delegate to Lua runtime
		var sessionID string
		if h.backendLookup != nil {
			if b := h.backendLookup.GetBackendForConnection(connectionID); b != nil {
				sessionID = b.GetSessionID()
			}
		}

		if sessionID == "" {
			return &Response{Error: "session context required for path variables"}, nil
		}

		err := h.pathVariableHandler.HandleFrontendCreate(sessionID, id, msg.ParentID, msg.Properties)
		if err != nil {
			h.Log(0, "Error, handleCreate: %s", err.Error())
			return &Response{Error: err.Error()}, nil
		}
	}

	// Auto-watch unless nowatch is set
	if !msg.NoWatch && h.backendLookup != nil {
		if b := h.backendLookup.GetBackendForConnection(connectionID); b != nil {
			b.Watch(id, connectionID)
		}
	}

	// No response needed - updates are sent via the normal change detection mechanism
	return &Response{}, nil
}

// handleDestroy processes a destroy message.
// Destroys the variable and all descendants in the backend, then notifies
// all watchers (including the originator) for each destroyed variable.
func (h *Handler) handleDestroy(connectionID string, data json.RawMessage) (*Response, error) {
	var msg DestroyMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	if h.backendLookup == nil {
		return &Response{}, nil
	}
	b := h.backendLookup.GetBackendForConnection(connectionID)
	if b == nil {
		return &Response{}, nil
	}

	// Destroy variable and all descendants; returns IDs children-first.
	// DestroyVariable clears watcher maps, so we send notifications
	// to the originator (who requested the destroy). Other watchers
	// of descendant variables are an edge case — in practice only the
	// originating connection watches frontend-created variables.
	destroyed := b.DestroyVariable(msg.VarID)

	// CRC: crc-ProtocolHandler.md | Seq: seq-destroy-variable.md | R112
	// Queue destroy notifications through batcher so they coalesce into
	// a single outgoing WebSocket frame instead of N individual frames.
	for _, varID := range destroyed {
		destroyNotif, _ := NewMessage(MsgDestroy, DestroyMessage{VarID: varID})
		if h.queuer != nil {
			h.Log(0, "DESTROY: using queuer for var %d", varID)
			h.queuer.Queue(destroyNotif, []string{connectionID})
		} else {
			h.Log(0, "DESTROY: queuer is nil, sending directly for var %d", varID)
			h.sender.Send(connectionID, destroyNotif)
		}
	}

	return &Response{}, nil
}

// handleUpdate processes an update message.
// CRC: crc-ProtocolHandler.md
// Sequence: seq-relay-message.md
func (h *Handler) handleUpdate(connectionID string, data json.RawMessage) (*Response, error) {
	var msg UpdateMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	// Get backend for this connection
	var b backend.Backend
	if h.backendLookup != nil {
		b = h.backendLookup.GetBackendForConnection(connectionID)
	}

	// Check if variable is inactive
	if b != nil && b.IsInactive(msg.VarID) {
		// Silently ignore updates to inactive variables
		return &Response{}, nil
	}

	// Handle inactive property
	if inactive, ok := msg.Properties["inactive"]; ok && b != nil {
		b.SetInactive(msg.VarID, inactive != "")
	}

	if h.pathVariableHandler != nil {
		var sessionID string
		if b != nil {
			sessionID = b.GetSessionID()
		}
		if sessionID == "" {
			return &Response{Error: "session context required for path variables"}, nil
		}
		if err := h.pathVariableHandler.HandleFrontendUpdate(sessionID, msg.VarID, msg.Value, msg.Properties); err != nil {
			h.Log(0, "ERROR, handleUpdate: backend update failed for var %d: %v", msg.VarID, err)
			return &Response{Error: err.Error()}, nil
		}
	}

	return &Response{}, nil
}

// handleWatch processes a watch message.
func (h *Handler) handleWatch(connectionID string, data json.RawMessage) (*Response, error) {
	var msg WatchMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	var result backend.WatchResult
	var b backend.Backend
	if h.backendLookup != nil {
		if b = h.backendLookup.GetBackendForConnection(connectionID); b != nil {
			result = b.Watch(msg.VarID, connectionID)
		}
	}

	if b == nil {
		return nil, fmt.Errorf("no backend for connection %s", connectionID)
	}

	v := b.GetTracker().GetVariable(msg.VarID)
	if v == nil {
		return nil, fmt.Errorf("variable %d not found", msg.VarID)
	}

	// Send current value immediately
	//props := v.Properties
	//h.Log(2, "handleWatch: sending update for var %d, type=%s, viewdefs=%d chars", msg.VarID, props["type"], len(props["viewdefs"]))
	val := v.WrapperJSON
	if val == nil {
		val = v.ValueJSON
	}
	if val != nil {
		b.GetTracker().ChangeAll(v.ID)
	}
	resp := &Response{}
	if result.ShouldForward {
		// For bound variables, indicate that watch should be forwarded to backend
		resp.Result = map[string]bool{"forward": true}
	}
	return resp, nil
}

// handleUnwatch processes an unwatch message.
func (h *Handler) handleUnwatch(connectionID string, data json.RawMessage) (*Response, error) {
	var msg WatchMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	var result backend.UnwatchResult
	if h.backendLookup != nil {
		if b := h.backendLookup.GetBackendForConnection(connectionID); b != nil {
			result = b.Unwatch(msg.VarID, connectionID)
		}
	}

	resp := &Response{}
	if result.ShouldForward {
		// For bound variables, indicate that unwatch should be forwarded to backend
		resp.Result = map[string]bool{"forward": true}
	}

	return resp, nil
}

// SendError sends an error message to a connection.
// Routes through queuer when available to maintain message ordering.
func (h *Handler) SendError(connectionID string, varID int64, description string) error {
	msg, err := NewMessage(MsgError, ErrorMessage{
		VarID:       varID,
		Description: description,
	})
	if err != nil {
		return err
	}
	if h.queuer != nil {
		h.queuer.Queue(msg, []string{connectionID})
		return nil
	}
	return h.sender.Send(connectionID, msg)
}
