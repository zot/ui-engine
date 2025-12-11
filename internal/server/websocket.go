// Package server implements the UI server communication layer.
// CRC: crc-WebSocketEndpoint.md
// Spec: interfaces.md
package server

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/zot/ui/internal/protocol"
	"github.com/zot/ui/internal/session"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// WebSocketEndpoint handles WebSocket connections.
type WebSocketEndpoint struct {
	connections     map[string]*websocket.Conn // connectionID -> conn
	sessionBindings map[string]string          // connectionID -> sessionID
	reconnectTokens map[string]string          // sessionID -> token
	sessions        *session.Manager
	handler         *protocol.Handler
	verbosity       int
	mu              sync.RWMutex
}

// NewWebSocketEndpoint creates a new WebSocket endpoint.
func NewWebSocketEndpoint(sessions *session.Manager, handler *protocol.Handler) *WebSocketEndpoint {
	return &WebSocketEndpoint{
		connections:     make(map[string]*websocket.Conn),
		sessionBindings: make(map[string]string),
		reconnectTokens: make(map[string]string),
		sessions:        sessions,
		handler:         handler,
	}
}

// SetVerbosity sets the verbosity level for connection logging.
func (ws *WebSocketEndpoint) SetVerbosity(level int) {
	ws.verbosity = level
}

// HandleWebSocket handles incoming WebSocket connections.
func (ws *WebSocketEndpoint) HandleWebSocket(w http.ResponseWriter, r *http.Request, sessionID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	connectionID := generateConnectionID()

	ws.mu.Lock()
	ws.connections[connectionID] = conn
	ws.sessionBindings[connectionID] = sessionID
	ws.mu.Unlock()

	// Log connection event (verbosity level 1)
	if ws.verbosity >= 1 {
		log.Printf("[v1] WebSocket connected: session=%s conn=%s", sessionID, connectionID)
	}

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
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse and handle message
		msg, err := protocol.ParseMessage(message)
		if err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		resp, err := ws.handler.HandleMessage(connectionID, msg)
		if err != nil {
			log.Printf("Failed to handle message: %v", err)
			ws.handler.SendError(connectionID, 0, err.Error())
			continue
		}

		// Send response if there's a result or error
		if resp != nil && (resp.Result != nil || resp.Error != "" || len(resp.Pending) > 0) {
			ws.sendResponse(connectionID, resp)
		}
	}
}

// sendResponse sends a response to a connection.
func (ws *WebSocketEndpoint) sendResponse(connectionID string, resp *protocol.Response) error {
	ws.mu.RLock()
	conn, ok := ws.connections[connectionID]
	ws.mu.RUnlock()

	if !ok {
		return nil
	}

	return conn.WriteJSON(resp)
}

// onDisconnect handles connection close.
func (ws *WebSocketEndpoint) onDisconnect(connectionID string) {
	ws.mu.Lock()
	sessionID := ws.sessionBindings[connectionID]
	delete(ws.connections, connectionID)
	delete(ws.sessionBindings, connectionID)
	ws.mu.Unlock()

	// Log disconnection event (verbosity level 1)
	if ws.verbosity >= 1 {
		log.Printf("[v1] WebSocket disconnected: session=%s conn=%s", sessionID, connectionID)
	}

	// Notify session
	if sess, ok := ws.sessions.GetSession(sessionID); ok {
		sess.RemoveConnection(connectionID)
	}
}

// Send sends a message to a specific connection.
func (ws *WebSocketEndpoint) Send(connectionID string, msg *protocol.Message) error {
	ws.mu.RLock()
	conn, ok := ws.connections[connectionID]
	ws.mu.RUnlock()

	if !ok {
		return nil
	}

	data, err := msg.Encode()
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// Broadcast sends a message to all connections in a session.
func (ws *WebSocketEndpoint) Broadcast(sessionID string, msg *protocol.Message) error {
	ws.mu.RLock()
	var conns []*websocket.Conn
	for connID, sessID := range ws.sessionBindings {
		if sessID == sessionID {
			if conn, ok := ws.connections[connID]; ok {
				conns = append(conns, conn)
			}
		}
	}
	ws.mu.RUnlock()

	data, err := msg.Encode()
	if err != nil {
		return err
	}

	for _, conn := range conns {
		conn.WriteMessage(websocket.TextMessage, data)
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
