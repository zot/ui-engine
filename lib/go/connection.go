// CRC: crc-BackendConnection.md
// Spec: libraries.md
// Package uiclient provides a client library for connecting to UI server.
package uiclient

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

// Connection represents a connection to the UI server.
type Connection struct {
	conn          net.Conn
	sessionID     string
	rootVariableID int64
	connected     bool
	messageQueue  []Message
	onClose       func()
	mu            sync.RWMutex
	readMu        sync.Mutex
	writeMu       sync.Mutex
}

// Message represents a protocol message.
type Message struct {
	Type       string                 `json:"type"`
	ID         int64                  `json:"id,omitempty"`
	ParentID   int64                  `json:"parentId,omitempty"`
	Value      interface{}            `json:"value,omitempty"`
	Properties map[string]string      `json:"properties,omitempty"`
	VarIDs     []int64                `json:"varIds,omitempty"`
	ObjIDs     []int64                `json:"objIds,omitempty"`
	Wait       string                 `json:"wait,omitempty"`
	Nowatch    bool                   `json:"nowatch,omitempty"`
	Unbound    bool                   `json:"unbound,omitempty"`
}

// Response represents a protocol response.
type Response struct {
	Result  interface{} `json:"result,omitempty"`
	Pending []Message   `json:"pending,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewConnection creates a new UI server connection.
func NewConnection() *Connection {
	return &Connection{
		messageQueue: make([]Message, 0),
	}
}

// Connect establishes connection to the UI server.
// socketPath is the Unix socket or named pipe path.
func (c *Connection) Connect(socketPath string) error {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()

	return nil
}

// Disconnect closes the connection.
func (c *Connection) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	c.connected = false
	if c.onClose != nil {
		c.onClose()
	}

	return c.conn.Close()
}

// IsConnected returns the connection state.
func (c *Connection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// OnClose registers a callback for connection close.
func (c *Connection) OnClose(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onClose = fn
}

// Send sends a message to the UI server.
func (c *Connection) Send(msg *Message) (*Response, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	// Encode message
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// Write length prefix (4 bytes, big-endian)
	length := uint32(len(data))
	if err := binary.Write(c.conn, binary.BigEndian, length); err != nil {
		return nil, err
	}

	// Write message
	if _, err := c.conn.Write(data); err != nil {
		return nil, err
	}

	// Read response
	return c.readResponse()
}

// readResponse reads a response from the server.
func (c *Connection) readResponse() (*Response, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	// Read length prefix
	var length uint32
	if err := binary.Read(c.conn, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// Read message
	data := make([]byte, length)
	if _, err := io.ReadFull(c.conn, data); err != nil {
		return nil, err
	}

	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// SetRootValue initializes variable 1 with a root value.
func (c *Connection) SetRootValue(value interface{}) error {
	_, err := c.Send(&Message{
		Type:  "update",
		ID:    1,
		Value: value,
	})
	return err
}

// Create creates a new variable.
func (c *Connection) Create(parentID int64, value interface{}, props map[string]string) (int64, error) {
	resp, err := c.Send(&Message{
		Type:       "create",
		ParentID:   parentID,
		Value:      value,
		Properties: props,
	})
	if err != nil {
		return 0, err
	}

	if resp.Error != "" {
		return 0, fmt.Errorf(resp.Error)
	}

	// Result should contain the new variable ID
	if result, ok := resp.Result.(map[string]interface{}); ok {
		if id, ok := result["id"].(float64); ok {
			return int64(id), nil
		}
	}

	return 0, fmt.Errorf("unexpected response format")
}

// Update updates a variable.
func (c *Connection) Update(id int64, value interface{}, props map[string]string) error {
	msg := &Message{
		Type: "update",
		ID:   id,
	}
	if value != nil {
		msg.Value = value
	}
	if props != nil {
		msg.Properties = props
	}

	_, err := c.Send(msg)
	return err
}

// Destroy destroys a variable.
func (c *Connection) Destroy(id int64) error {
	_, err := c.Send(&Message{
		Type: "destroy",
		ID:   id,
	})
	return err
}

// Watch subscribes to variable updates.
func (c *Connection) Watch(id int64) error {
	_, err := c.Send(&Message{
		Type: "watch",
		ID:   id,
	})
	return err
}

// Unwatch unsubscribes from variable updates.
func (c *Connection) Unwatch(id int64) error {
	_, err := c.Send(&Message{
		Type: "unwatch",
		ID:   id,
	})
	return err
}

// Get retrieves variable values.
func (c *Connection) Get(ids ...int64) ([]interface{}, error) {
	resp, err := c.Send(&Message{
		Type:   "get",
		VarIDs: ids,
	})
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}

	if result, ok := resp.Result.([]interface{}); ok {
		return result, nil
	}

	return nil, fmt.Errorf("unexpected response format")
}

// Poll retrieves pending messages.
func (c *Connection) Poll(wait string) (*Response, error) {
	return c.Send(&Message{
		Type: "poll",
		Wait: wait,
	})
}

// GetSessionID returns the current session ID.
func (c *Connection) GetSessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sessionID
}

// SetSessionID sets the session ID.
func (c *Connection) SetSessionID(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = id
}
