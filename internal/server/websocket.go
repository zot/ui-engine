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
type wsConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

// WebSocketEndpoint handles WebSocket connections.
type WebSocketEndpoint struct {
	config          *config.Config
	connections     map[string]*wsConn // connectionID -> conn
	sessionBindings map[string]string  // connectionID -> sessionID
	reconnectTokens map[string]string  // sessionID -> token
	sessionSvc      map[string]ChanSvc // sessionID -> executor (serializes session operations)
	sessions        *SessionManager
	handler         *protocol.Handler
	afterBatch      AfterBatchCallback // Called after each message to detect changes
	onDisconnectCb  DisconnectCallback // Called when a connection disconnects
	mu              sync.RWMutex
}

// NewWebSocketEndpoint creates a new WebSocket endpoint.
func NewWebSocketEndpoint(cfg *config.Config, sessions *SessionManager, handler *protocol.Handler) *WebSocketEndpoint {
	return &WebSocketEndpoint{
		config:          cfg,
		connections:     make(map[string]*wsConn),
		sessionBindings: make(map[string]string),
		reconnectTokens: make(map[string]string),
		sessionSvc:      make(map[string]ChanSvc),
		sessions:        sessions,
		handler:         handler,
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

// getOrCreateSvc returns the executor for a session, creating if needed.
func (ws *WebSocketEndpoint) getOrCreateSvc(sessionID string) ChanSvc {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if svc, ok := ws.sessionSvc[sessionID]; ok {
		return svc
	}

	svc := make(ChanSvc)
	ws.sessionSvc[sessionID] = svc
	RunSvc(svc)
	return svc
}

// cleanupSessionSvc closes and removes a session's executor.
func (ws *WebSocketEndpoint) cleanupSessionSvc(sessionID string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if svc, ok := ws.sessionSvc[sessionID]; ok {
		close(svc)
		delete(ws.sessionSvc, sessionID)
	}
}

// ExecuteInSession executes a function within a session's executor.
// This serializes the execution with WebSocket message processing for the session.
// AfterBatch is called after execution to detect and push any changes,
// but only if there are active browser connections to receive the updates.
// Returns the result and any error from the function.
func (ws *WebSocketEndpoint) ExecuteInSession(sessionID string, fn func() (interface{}, error)) (interface{}, error) {
	svc := ws.getOrCreateSvc(sessionID)
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

// HandleWebSocket handles incoming WebSocket connections.
func (ws *WebSocketEndpoint) HandleWebSocket(w http.ResponseWriter, r *http.Request, sessionID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.Log(0, "WebSocket upgrade failed: %v", err)
		return
	}

	connectionID := generateConnectionID()

	ws.mu.Lock()
	ws.connections[connectionID] = &wsConn{conn: conn}
	ws.sessionBindings[connectionID] = sessionID
	ws.mu.Unlock()

	// Log connection event (verbosity level 1)
	ws.Log(1, "WebSocket connected: session=%s conn=%s", sessionID, connectionID)

	// Add connection to session
	if sess, ok := ws.sessions.GetSession(sessionID); ok {
		sess.AddConnection(connectionID)
	}

	// Handle messages
	go ws.readPump(connectionID, conn)
}

// readPump reads messages from a WebSocket connection.
func (ws *WebSocketEndpoint) readPump(connectionID string, conn *websocket.Conn) {
	defer func() {
		ws.onDisconnect(connectionID)
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ws.Log(0, "WebSocket error: %v", err)
			}
			break
		}

		// Get session for this connection
		ws.mu.RLock()
		sessionID := ws.sessionBindings[connectionID]
		ws.mu.RUnlock()

		if sessionID == "" {
			continue
		}

		// Queue message processing through session's executor
		svc := ws.getOrCreateSvc(sessionID)
		Svc(svc, func() {
			ws.processMessage(connectionID, sessionID, message)
		})
	}
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

	// For non-user events, start debounce timer BEFORE processing
	// so timer runs concurrently with message handling
	if !userEvent && session != nil {
		session.EnsureDebounceStarted()
	}

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
func (ws *WebSocketEndpoint) onDisconnect(connectionID string) {
	ws.mu.Lock()
	sessionID := ws.sessionBindings[connectionID]
	delete(ws.connections, connectionID)
	delete(ws.sessionBindings, connectionID)
	ws.mu.Unlock()

	// Log disconnection event (verbosity level 1)
	ws.Log(1, "WebSocket disconnected: session=%s conn=%s", sessionID, connectionID)

	// Notify session
	if sess, ok := ws.sessions.GetSession(sessionID); ok {
		sess.RemoveConnection(connectionID)
	}

	// Notify disconnect callback (used to clear sent-tracking for page refresh)
	if ws.onDisconnectCb != nil && sessionID != "" {
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

// Broadcast sends a message to all connections in a session.
func (ws *WebSocketEndpoint) Broadcast(sessionID string, msg *protocol.Message) error {
	ws.mu.RLock()
	var conns []*wsConn
	for connID, sessID := range ws.sessionBindings {
		if sessID == sessionID {
			if wc, ok := ws.connections[connID]; ok {
				conns = append(conns, wc)
			}
		}
	}
	ws.mu.RUnlock()

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

// GetSessionID returns the session ID for a connection.
func (ws *WebSocketEndpoint) GetSessionID(connectionID string) (string, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	sessionID, ok := ws.sessionBindings[connectionID]
	return sessionID, ok
}

// GetSessionIDForConnection returns the session ID for a connection.
// Returns empty string if connection is not found.
func (ws *WebSocketEndpoint) GetSessionIDForConnection(connectionID string) string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.sessionBindings[connectionID]
}

// HasConnectionsForSession returns true if the session has any active connections.
func (ws *WebSocketEndpoint) HasConnectionsForSession(sessionID string) bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	for _, sessID := range ws.sessionBindings {
		if sessID == sessionID {
			return true
		}
	}
	return false
}

// IsSessionReconnectable checks if a session can be rejoined.
// A session can be rejoined if it exists and hasn't timed out.
func (ws *WebSocketEndpoint) IsSessionReconnectable(sessionID string) bool {
	return ws.sessions.SessionExists(sessionID)
}

// GenerateReconnectToken creates a token for validating reconnection.
func (ws *WebSocketEndpoint) GenerateReconnectToken(sessionID string) string {
	token := generateToken()
	ws.mu.Lock()
	ws.reconnectTokens[sessionID] = token
	ws.mu.Unlock()
	return token
}

// ValidateReconnectToken validates a reconnection token.
func (ws *WebSocketEndpoint) ValidateReconnectToken(sessionID, token string) bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	expected, ok := ws.reconnectTokens[sessionID]
	return ok && expected == token
}

func generateConnectionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "conn-" + hex.EncodeToString(bytes)
}

func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
