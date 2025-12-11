// CRC: crc-ProtocolHandler.md
// Spec: protocol.md
package protocol

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

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

// Handler processes protocol messages.
type Handler struct {
	store     *variable.Store
	watches   *variable.WatchManager
	sender    MessageSender
	pending   PendingQueuer
	verbosity int
}

// NewHandler creates a new protocol handler.
func NewHandler(store *variable.Store, watches *variable.WatchManager, sender MessageSender) *Handler {
	return &Handler{
		store:   store,
		watches: watches,
		sender:  sender,
	}
}

// SetPendingQueuer sets the pending queue manager.
func (h *Handler) SetPendingQueuer(pending PendingQueuer) {
	h.pending = pending
}

// SetVerbosity sets the verbosity level for message logging.
func (h *Handler) SetVerbosity(level int) {
	h.verbosity = level
}

// HandleMessage processes an incoming protocol message.
func (h *Handler) HandleMessage(connectionID string, msg *Message) (*Response, error) {
	// Log message (verbosity level 2)
	if h.verbosity >= 2 {
		log.Printf("[v2] Message: type=%s from=%s", msg.Type, connectionID)
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
	var msg CreateMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	id, err := h.store.Create(variable.CreateOptions{
		ParentID:   msg.ParentID,
		Value:      msg.Value,
		Properties: msg.Properties,
		NoWatch:    msg.NoWatch,
		Unbound:    msg.Unbound,
	})
	if err != nil {
		return &Response{Error: err.Error()}, nil
	}

	// Auto-watch unless nowatch is set
	if !msg.NoWatch {
		h.watches.Watch(id, connectionID)
	}

	return &Response{
		Result: CreateResponse{ID: id},
	}, nil
}

// handleDestroy processes a destroy message.
func (h *Handler) handleDestroy(connectionID string, data json.RawMessage) (*Response, error) {
	var msg DestroyMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	// Get watchers before destroying
	watchers := h.watches.GetWatchers(msg.VarID)

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
func (h *Handler) handleUpdate(connectionID string, data json.RawMessage) (*Response, error) {
	var msg UpdateMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	// Check if variable is inactive
	if h.watches.IsInactive(msg.VarID) {
		// Silently ignore updates to inactive variables
		return &Response{}, nil
	}

	// Handle inactive property
	if inactive, ok := msg.Properties["inactive"]; ok {
		h.watches.SetInactive(msg.VarID, inactive != "")
	}

	if err := h.store.Update(msg.VarID, msg.Value, msg.Properties); err != nil {
		return &Response{Error: err.Error()}, nil
	}

	// Notify watchers of update
	watchers := h.watches.GetWatchers(msg.VarID)
	if len(watchers) > 0 {
		v, ok := h.store.Get(msg.VarID)
		if ok {
			updateMsg, _ := NewMessage(MsgUpdate, UpdateMessage{
				VarID:      msg.VarID,
				Value:      v.GetValue(),
				Properties: v.GetProperties(),
			})
			for _, watcherID := range watchers {
				if watcherID != connectionID {
					h.sender.Send(watcherID, updateMsg)
				}
			}
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

	result := h.watches.Watch(msg.VarID, connectionID)

	// Send current value immediately
	v, ok := h.store.Get(msg.VarID)
	if ok {
		updateMsg, _ := NewMessage(MsgUpdate, UpdateMessage{
			VarID:      msg.VarID,
			Value:      v.GetValue(),
			Properties: v.GetProperties(),
		})
		h.sender.Send(connectionID, updateMsg)
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

	result := h.watches.Unwatch(msg.VarID, connectionID)

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
			variables = append(variables, VariableData{
				ID:         v.ID,
				Value:      v.GetValue(),
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
			objects = append(objects, ObjectData{
				ID:    id,
				Value: v.GetValue(),
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
