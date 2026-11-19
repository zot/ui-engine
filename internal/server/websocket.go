// Package server implements the UI server communication layer.
// CRC: crc-WebSocketEndpoint.md
// Spec: interfaces.md
package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zot/ui-engine/internal/config"
	"github.com/zot/ui-engine/internal/protocol"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// AfterBatchCallback is called after processing a message batch to trigger change detection.
// userEvent indicates if the batch was triggered by user interaction (immediate flush needed).
type AfterBatchCallback func(sessionID string, userEvent bool)

// DisconnectCallback is called when a connection disconnects.
// Used to clear sent-tracking so reconnections resync state.
type DisconnectCallback func(sessionID string)

// wsConn wraps a websocket connection with a write mutex.
// gorilla/websocket does not support concurrent writes.
// session is the Session the connection was opened on, set once by register when the
// session exists and never changed; it is the only connection-to-session record.
// CRC: crc-WebSocketEndpoint.md | R244, R245
type wsConn struct {
	conn    *websocket.Conn
	session *Session
	writeMu sync.Mutex
}

// WebSocketEndpoint handles WebSocket connections.
type WebSocketEndpoint struct {
	config         *config.Config
	connections    map[string]*wsConn // connectionID -> conn
	sessions       *SessionManager
	handler        *protocol.Handler
	afterBatch     AfterBatchCallback // Called after each message to detect changes
	onDisconnectCb DisconnectCallback // Called when a connection disconnects
	mu             sync.RWMutex
}

// NewWebSocketEndpoint creates a new WebSocket endpoint.
func NewWebSocketEndpoint(cfg *config.Config, sessions *SessionManager, handler *protocol.Handler) *WebSocketEndpoint {
	return &WebSocketEndpoint{
		config:      cfg,
		connections: make(map[string]*wsConn),
		sessions:    sessions,
		handler:     handler,
	}
}

// Log logs a message via the config.
func (ws *WebSocketEndpoint) Log(level int, format string, args ...interface{}) {
	ws.config.Log(level, format, args...)
}

// SetAfterBatch sets the callback for change detection after message processing.
func (ws *WebSocketEndpoint) SetAfterBatch(callback AfterBatchCallback) {
	ws.afterBatch = callback
}

// SetOnDisconnect sets the callback for when a connection disconnects.
// This is used to clear sent-tracking so page refreshes resync all state.
func (ws *WebSocketEndpoint) SetOnDisconnect(callback DisconnectCallback) {
	ws.onDisconnectCb = callback
}

// getSession returns the session for the given ID, or nil if not found.
func (ws *WebSocketEndpoint) getSession(sessionID string) *Session {
	sess, _ := ws.sessions.GetSession(sessionID)
	return sess
}

// executor returns the session's executor, or nil when there is no such session. It
// never creates one: work for a session that is gone has nowhere to run.
// CRC: crc-WebSocketEndpoint.md | Seq: seq-session-executor.md#3.1.1 | R228
func (ws *WebSocketEndpoint) executor(sessionID string) *ChanSvc {
	if sess := ws.getSession(sessionID); sess != nil {
		return sess.Executor()
	}
	return nil
}

// ExecuteInSession executes a function within a session's executor.
// This serializes the execution with WebSocket message processing for the session.
// AfterBatch is called after execution to detect and push any changes,
// but only if there are active browser connections to receive the updates.
// Returns the result and any error from the function.
// With no such session it returns an error; when the session is destroyed while the
// call waits, ErrStopped.
// CRC: crc-WebSocketEndpoint.md | Seq: seq-session-executor.md#3.1.2 | R228
func (ws *WebSocketEndpoint) ExecuteInSession(sessionID string, fn func() (interface{}, error)) (interface{}, error) {
	svc := ws.executor(sessionID)
	if svc == nil {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	return SvcSync(svc, func() (interface{}, error) {
		result, err := fn()
		// Trigger change detection after execution, but only if there are connections
		// This prevents marking viewdefs as "sent" before any browser is connected
		// Use userEvent=false since this is backend-triggered (not user interaction)
		if ws.afterBatch != nil && ws.HasConnectionsForSession(sessionID) {
			ws.afterBatch(sessionID, false)
		}
		return result, err
	})
}

// ExecuteInSessionAsync is a fire-and-forget variant of ExecuteInSession.
// It queues execution through ChanSvc using Svc (async) instead of SvcSync (blocking).
// AfterBatch is called after execution to detect and push any changes.
// CRC: crc-LuaSession.md
// Seq: seq-session-timer.md
// With no such session the work is dropped.
// CRC: crc-WebSocketEndpoint.md | Seq: seq-session-executor.md#3.1.2 | R228
func (ws *WebSocketEndpoint) ExecuteInSessionAsync(sessionID string, fn func() (interface{}, error)) {
	svc := ws.executor(sessionID)
	if svc == nil {
		return
	}
	Svc(svc, func() {
		fn()
		if ws.afterBatch != nil && ws.HasConnectionsForSession(sessionID) {
			ws.afterBatch(sessionID, false)
		}
	})
}

// ExecuteInSessionPassive queues fn on the session executor without waiting and
// without running afterBatch afterward. The passive twin of ExecuteInSessionAsync.
// With no such session the work is dropped.
// CRC: crc-WebSocketEndpoint.md | Seq: seq-session-ref.md#1.2.3, seq-session-executor.md#3.1.2 | R159, R228
func (ws *WebSocketEndpoint) ExecuteInSessionPassive(sessionID string, fn func()) {
	svc := ws.executor(sessionID)
	if svc == nil {
		return
	}
	Svc(svc, fn)
}

// RunAfterBatch runs change detection and pushes changes now, when a browser is
// connected. Call it only on the session executor.
// CRC: crc-WebSocketEndpoint.md | R163
func (ws *WebSocketEndpoint) RunAfterBatch(sessionID string) {
	if ws.afterBatch != nil && ws.HasConnectionsForSession(sessionID) {
		ws.afterBatch(sessionID, false)
	}
}

// HandleWebSocket handles incoming WebSocket connections.
func (ws *WebSocketEndpoint) HandleWebSocket(w http.ResponseWriter, r *http.Request, sessionID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.Log(0, "WebSocket upgrade failed: %v", err)
		return
	}

	connectionID := generateConnectionID()
	wc := ws.register(connectionID, conn, sessionID)

	// Handle messages
	go ws.readPump(connectionID, wc)
}

// register is the binding step of a new connection: the session is looked up once, and
// when it exists the connection holds it and the session records the connection. A
// connection to a session that does not exist holds nothing.
// CRC: crc-WebSocketEndpoint.md | R244, R245
func (ws *WebSocketEndpoint) register(connectionID string, conn *websocket.Conn, sessionID string) *wsConn {
	wc := &wsConn{conn: conn, session: ws.getSession(sessionID)}
	ws.mu.Lock()
	ws.connections[connectionID] = wc
	ws.mu.Unlock()

	// Log connection event (verbosity level 1)
	ws.Log(1, "WebSocket connected: session=%s conn=%s", sessionID, connectionID)

	if wc.session != nil {
		wc.session.AddConnection(connectionID)
	}
	return wc
}

// readPump reads messages from a WebSocket connection.
func (ws *WebSocketEndpoint) readPump(connectionID string, wc *wsConn) {
	defer func() {
		ws.onDisconnect(connectionID)
		wc.conn.Close()
	}()

	for {
		_, message, err := wc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ws.Log(0, "WebSocket error: %v", err)
			}
			break
		}

		ws.postMessage(connectionID, wc, message)
	}
}

// postMessage posts one message to the connection's session's executor and reports
// whether it did. A connection with no session posts nothing. The session's executor
// drops it once the session is destroyed (R228).
// CRC: crc-WebSocketEndpoint.md | Seq: seq-session-executor.md#3.1.2 | R246
func (ws *WebSocketEndpoint) postMessage(connectionID string, wc *wsConn, message []byte) bool {
	if wc.session == nil {
		return false
	}
	sessionID := wc.session.ID
	Svc(wc.session.Executor(), func() {
		ws.processMessage(connectionID, sessionID, message)
	})
	return true
}

// processMessage handles one or more messages within the session's executor.
// Supports single messages, batched arrays, and batch wrapper with userEvent flag.
// Spec: protocol.md - Message batching with userEvent flag
func (ws *WebSocketEndpoint) processMessage(connectionID, sessionID string, message []byte) {
	// Recover from panics to prevent server crashes
	defer func() {
		if r := recover(); r != nil {
			ws.Log(0, "PANIC in processMessage: %v", r)
			ws.handler.SendError(connectionID, 0, fmt.Sprintf("internal error: %v", r))
		}
	}()

	// Parse message batch and extract userEvent flag
	msgs, userEvent, err := protocol.ParseMessages(message)
	if err != nil {
		ws.Log(0, "Failed to parse message: %v", err)
		return
	}

	// Get session for debounce
	session := ws.getSession(sessionID)
	session.batchCount += 1
	count := session.batchCount
	evtMsg := "USER EVENT"
	if !userEvent {
		evtMsg = "NO USER EVENT"
	}
	ws.Log(4, "[IN] BATCH %d (%s)", count, evtMsg)

	// Process each message in the batch
	for _, msg := range msgs {
		resp, err := ws.handler.HandleMessage(connectionID, msg)
		if err != nil {
			ws.Log(0, "Failed to handle message: %v", err)
			ws.handler.SendError(connectionID, 0, err.Error())
			continue
		}

		// Send response if there's an error
		// Note: create no longer returns a response (frontend-vended IDs)
		if resp != nil && resp.Error != "" {
			ws.sendResponse(connectionID, resp)
		}
	}

	// Trigger change detection once after processing all messages
	// Pass userEvent flag to determine immediate vs debounced response
	if ws.afterBatch != nil {
		ws.afterBatch(sessionID, userEvent)
	}
}

// sendResponse sends a response to a connection.
func (ws *WebSocketEndpoint) sendResponse(connectionID string, resp *protocol.Response) error {
	ws.mu.RLock()
	wc, ok := ws.connections[connectionID]
	ws.mu.RUnlock()

	if !ok {
		return nil
	}

	// Log response
	if ws.config.Verbosity() >= 4 {
		if respJson, err := json.Marshal(resp); err != nil {
			ws.Log(4, "[OUT] RESPONSE: to=%s data=%+v", connectionID, respJson)
		}
	} else {
		ws.Log(2, "[OUT] RESPONSE: to=%s", connectionID)
	}

	wc.writeMu.Lock()
	defer wc.writeMu.Unlock()
	return wc.conn.WriteJSON(resp)
}

// onDisconnect handles connection close.
// It forgets the connection and removes it from its session, undoing both directions of
// the relationship together.
// CRC: crc-WebSocketEndpoint.md | R244, R245
func (ws *WebSocketEndpoint) onDisconnect(connectionID string) {
	ws.mu.Lock()
	wc := ws.connections[connectionID]
	delete(ws.connections, connectionID)
	ws.mu.Unlock()
	if wc == nil {
		return
	}
	sessionID := ""
	if wc.session != nil {
		sessionID = wc.session.ID
	}

	// Log disconnection event (verbosity level 1)
	ws.Log(1, "WebSocket disconnected: session=%s conn=%s", sessionID, connectionID)

	if wc.session == nil {
		return
	}
	wc.session.RemoveConnection(connectionID)

	// Notify disconnect callback (used to clear sent-tracking for page refresh)
	if ws.onDisconnectCb != nil {
		ws.onDisconnectCb(sessionID)
	}
}

// Send sends a message to a specific connection.
func (ws *WebSocketEndpoint) Send(connectionID string, msg *protocol.Message) error {
	ws.mu.RLock()
	wc, ok := ws.connections[connectionID]
	ws.mu.RUnlock()

	if !ok {
		return nil
	}

	// Log message
	msgType := strings.ToUpper(string(msg.Type))
	if ws.config.Verbosity() >= 4 {
		ws.Log(4, "[OUT] %s: to=%s data=%s", msgType, connectionID, string(msg.Data))
	} else {
		ws.Log(2, "[OUT] %s: to=%s", msgType, connectionID)
	}

	data, err := msg.Encode()
	if err != nil {
		return err
	}

	wc.writeMu.Lock()
	defer wc.writeMu.Unlock()
	return wc.conn.WriteMessage(websocket.TextMessage, data)
}

// SendBatch sends multiple messages as a JSON array to a specific connection.
// Spec: protocol.md - Server sends batched messages as JSON arrays
func (ws *WebSocketEndpoint) SendBatch(connectionID string, msgs []*protocol.Message) error {
	ws.mu.RLock()
	wc, ok := ws.connections[connectionID]
	ws.mu.RUnlock()

	if !ok {
		return nil
	}

	// Log batch
	ws.Log(2, "[OUT] BATCH: to=%s count=%d", connectionID, len(msgs))

	// Encode as JSON array
	data, err := json.Marshal(msgs)
	if err != nil {
		return err
	}

	wc.writeMu.Lock()
	defer wc.writeMu.Unlock()
	return wc.conn.WriteMessage(websocket.TextMessage, data)
}

// Broadcast sends a message to all connections in a session. The session lists its own
// connections; a destroyed session is no longer found, so nothing is sent.
// CRC: crc-WebSocketEndpoint.md | R247
func (ws *WebSocketEndpoint) Broadcast(sessionID string, msg *protocol.Message) error {
	var conns []*wsConn
	if sess := ws.getSession(sessionID); sess != nil {
		ids := sess.ConnectionIDs()
		ws.mu.RLock()
		for _, connID := range ids {
			if wc, ok := ws.connections[connID]; ok {
				conns = append(conns, wc)
			}
		}
		ws.mu.RUnlock()
	}

	// Log message
	msgType := strings.ToUpper(string(msg.Type))
	if ws.config.Verbosity() >= 4 {
		ws.Log(4, "[OUT] %s: to=session:%s data=%s", msgType, sessionID, string(msg.Data))
	} else {
		ws.Log(2, "[OUT] %s: to=session:%s", msgType, sessionID)
	}

	data, err := msg.Encode()
	if err != nil {
		return err
	}

	for _, wc := range conns {
		wc.writeMu.Lock()
		wc.conn.WriteMessage(websocket.TextMessage, data)
		wc.writeMu.Unlock()
	}
	return nil
}

// IsConnected checks if a connection is active.
func (ws *WebSocketEndpoint) IsConnected(connectionID string) bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	_, ok := ws.connections[connectionID]
	return ok
}

// GetSessionIDForConnection returns the session ID for a connection.
// Returns empty string if connection is not found.
// CRC: crc-WebSocketEndpoint.md | R244
func (ws *WebSocketEndpoint) GetSessionIDForConnection(connectionID string) string {
	ws.mu.RLock()
	wc := ws.connections[connectionID]
	ws.mu.RUnlock()
	if wc == nil || wc.session == nil {
		return ""
	}
	return wc.session.ID
}

// HasConnectionsForSession returns true if the session has any active connections.
// It asks the session; a destroyed session is no longer found, so it has none.
// CRC: crc-WebSocketEndpoint.md | R162, R247
func (ws *WebSocketEndpoint) HasConnectionsForSession(sessionID string) bool {
	sess := ws.getSession(sessionID)
	return sess != nil && sess.GetConnectionCount() > 0
}

// IsSessionReconnectable checks if a session can be rejoined.
// A session can be rejoined if it exists and hasn't timed out.
// No reconnect token is issued or checked: existing and not timed out is the whole test.
// CRC: crc-WebSocketEndpoint.md | R248
func (ws *WebSocketEndpoint) IsSessionReconnectable(sessionID string) bool {
	return ws.sessions.SessionExists(sessionID)
}

func generateConnectionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "conn-" + hex.EncodeToString(bytes)
}
