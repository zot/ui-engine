// CRC: crc-ProtocolHandler.md
// Spec: protocol.md
package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zot/ui/internal/backend"
	"github.com/zot/ui/internal/config"
	"github.com/zot/ui/internal/variable"
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
	// Returns the variable ID, resolved value, and properties.
	HandleFrontendCreate(sessionID string, parentID int64, properties map[string]string) (int64, json.RawMessage, map[string]string, error)

	// HandleFrontendUpdate handles an update to a path-based variable from frontend.
	// Updates the backend object via the variable's path and returns error if any.
	HandleFrontendUpdate(sessionID string, varID int64, value json.RawMessage) error
}

// BackendLookup provides per-connection backend lookup.
// Used by the protocol handler to route watch operations to the correct session's backend.
type BackendLookup interface {
	// GetBackendForConnection returns the backend for a connection.
	// Returns nil if connection is not associated with a session.
	GetBackendForConnection(connectionID string) backend.Backend
}

// Handler processes protocol messages.
type Handler struct {
	config              *config.Config
	store               *variable.Store
	backendLookup       BackendLookup
	sender              MessageSender
	pending             PendingQueuer
	pathVariableHandler PathVariableHandler // For path-based frontend creates
}

// NewHandler creates a new protocol handler.
func NewHandler(cfg *config.Config, store *variable.Store, sender MessageSender) *Handler {
	return &Handler{
		config: cfg,
		store:  store,
		sender: sender,
	}
}

// SetBackendLookup sets the backend lookup for per-session watch operations.
func (h *Handler) SetBackendLookup(lookup BackendLookup) {
	h.backendLookup = lookup
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
	case MsgGet:
		return h.handleGet(msg.Data)
	case MsgGetObjects:
		return h.handleGetObjects(msg.Data)
	case MsgPoll:
		return h.handlePoll(connectionID, msg.Data)
	default:
		return nil, fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleCreate processes a create message.
func (h *Handler) handleCreate(connectionID string, data json.RawMessage) (*Response, error) {
	h.Log(2, "handleCreate 1, connection %s", connectionID)
	var msg CreateMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	var id int64
	var initialValue json.RawMessage
	var initialProps map[string]string
	var err error

	// Check if this is a path-based variable (has path property and parent)
	pathProp := ""
	if msg.Properties != nil {
		pathProp = msg.Properties["path"]
	}

	h.Log(2, "handleCreate 2")
	if pathProp != "" && msg.ParentID != 0 && h.pathVariableHandler != nil {
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

		id, initialValue, initialProps, err = h.pathVariableHandler.HandleFrontendCreate(sessionID, msg.ParentID, msg.Properties)
		if err != nil {
			h.Log(2, "handleCreate 2.1 ERROR: %s", err.Error())
			return &Response{Error: err.Error()}, nil
		}

		// Also create in UI server's store for tracking
		h.store.Create(variable.CreateOptions{
			ID:         id,
			ParentID:   msg.ParentID,
			Value:      initialValue,
			Properties: initialProps,
		})
	} else {
		// Regular variable: create in store
		initialProps = msg.Properties
		id, err = h.store.Create(variable.CreateOptions{
			ParentID:   msg.ParentID,
			Value:      msg.Value,
			Properties: initialProps,
			NoWatch:    msg.NoWatch,
			Unbound:    msg.Unbound,
		})
		if err != nil {
			h.Log(2, "handleCreate 2.2 ERROR: %s", err.Error())
			return &Response{Error: err.Error()}, nil
		}
	}

	h.Log(2, "handleCreate 3")
	// Auto-watch unless nowatch is set
	if !msg.NoWatch && h.backendLookup != nil {
		if b := h.backendLookup.GetBackendForConnection(connectionID); b != nil {
			b.Watch(id, connectionID)
		}
	}

	// Build response
	resp := &Response{
		Result: CreateResponse{ID: id},
	}

	// Include initial value as pending update if we have one
	if initialValue != nil || initialProps != nil {
		updateMsg, _ := NewMessage(MsgUpdate, UpdateMessage{
			VarID:      id,
			Value:      initialValue,
			Properties: initialProps,
		})
		resp.Pending = append(resp.Pending, *updateMsg)
	}

	h.Log(2, "handleCreate 4")
	return resp, nil
}

// handleDestroy processes a destroy message.
func (h *Handler) handleDestroy(connectionID string, data json.RawMessage) (*Response, error) {
	var msg DestroyMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	// Get watchers before destroying (via session's backend)
	var watchers []string
	if h.backendLookup != nil {
		if b := h.backendLookup.GetBackendForConnection(connectionID); b != nil {
			watchers = b.GetWatchers(msg.VarID)
		}
	}

	if err := h.store.Destroy(msg.VarID); err != nil {
		return &Response{Error: err.Error()}, nil
	}

	// Notify watchers of destruction
	destroyNotif, _ := NewMessage(MsgDestroy, DestroyMessage{VarID: msg.VarID})
	for _, watcherID := range watchers {
		if watcherID != connectionID {
			h.sender.Send(watcherID, destroyNotif)
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
		if err := h.pathVariableHandler.HandleFrontendUpdate(sessionID, msg.VarID, msg.Value); err != nil {
			h.Log(1, "handleUpdate: backend update failed for var %d: %v", msg.VarID, err)
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
	if h.backendLookup != nil {
		if b := h.backendLookup.GetBackendForConnection(connectionID); b != nil {
			result = b.Watch(msg.VarID, connectionID)
		}
	}

	// Send current value immediately
	v, ok := h.store.Get(msg.VarID)
	if ok {
		props := v.GetProperties()
		h.Log(2, "handleWatch: sending update for var %d, type=%s, viewdefs=%d chars", msg.VarID, props["type"], len(props["viewdefs"]))
		valBytes, _ := json.Marshal(v.GetValue())
		updateMsg, _ := NewMessage(MsgUpdate, UpdateMessage{
			VarID:      msg.VarID,
			Value:      valBytes,
			Properties: v.GetProperties(),
		})
		h.sender.Send(connectionID, updateMsg)
	} else {
		h.Log(2, "handleWatch: var %d not found in store!", msg.VarID)
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

// handleGet processes a get message.
func (h *Handler) handleGet(data json.RawMessage) (*Response, error) {
	var msg GetMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	variables := make([]VariableData, 0, len(msg.VarIDs))
	for _, id := range msg.VarIDs {
		v, ok := h.store.Get(id)
		if ok {
			valBytes, _ := json.Marshal(v.GetValue())
			variables = append(variables, VariableData{
				ID:         v.ID,
				Value:      valBytes,
				Properties: v.GetProperties(),
			})
		}
	}

	return &Response{
		Result: GetResponse{Variables: variables},
	}, nil
}

// handleGetObjects processes a getObjects message.
func (h *Handler) handleGetObjects(data json.RawMessage) (*Response, error) {
	var msg GetObjectsMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	objects := make([]ObjectData, 0, len(msg.ObjIDs))
	for _, id := range msg.ObjIDs {
		v, ok := h.store.Get(id)
		if ok {
			valBytes, _ := json.Marshal(v.GetValue())
			objects = append(objects, ObjectData{
				ID:    id,
				Value: valBytes,
			})
		}
	}

	return &Response{
		Result: GetObjectsResponse{Objects: objects},
	}, nil
}

// handlePoll processes a poll message for long-polling.
func (h *Handler) handlePoll(connectionID string, data json.RawMessage) (*Response, error) {
	var msg PollMessage
	if data != nil {
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, err
		}
	}

	// Parse wait duration if specified
	var waitDuration time.Duration
	if msg.Wait != "" {
		var err error
		waitDuration, err = time.ParseDuration(msg.Wait)
		if err != nil {
			return &Response{Error: "invalid wait duration"}, nil
		}
	}

	// Get pending messages from queue
	var pending []Message
	if h.pending != nil {
		messages := h.pending.Poll(connectionID, waitDuration)
		for _, m := range messages {
			if m != nil {
				pending = append(pending, *m)
			}
		}
	}

	return &Response{
		Pending: pending,
	}, nil
}

// SendError sends an error message to a connection.
func (h *Handler) SendError(connectionID string, varID int64, description string) error {
	msg, err := NewMessage(MsgError, ErrorMessage{
		VarID:       varID,
		Description: description,
	})
	if err != nil {
		return err
	}
	return h.sender.Send(connectionID, msg)
}
