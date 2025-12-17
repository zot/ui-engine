// Package session implements session management for the UI server.
// CRC: crc-Session.md, crc-SessionManager.md
// Spec: main.md (UI Server Architecture - Frontend Layer), interfaces.md
package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/zot/ui/internal/backend"
)

// Session represents a single user session.
// Session is part of the frontend layer - it routes messages to backend.
// CRC: crc-Session.md
type Session struct {
	ID            string
	AppVariableID int64               // Variable 1 - root app variable
	backend       backend.Backend     // Backend instance (LuaBackend or ProxiedBackend)
	connections   map[string]struct{} // connection IDs
	createdAt     time.Time
	lastActivity  time.Time
	mu            sync.RWMutex
}

// NewSession creates a new session with the given ID.
func NewSession(id string) *Session {
	now := time.Now()
	return &Session{
		ID:           id,
		connections:  make(map[string]struct{}),
		createdAt:    now,
		lastActivity: now,
	}
}

// GetID returns the session ID.
func (s *Session) GetID() string {
	return s.ID
}

// GetBackend returns the backend instance for this session.
func (s *Session) GetBackend() backend.Backend {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.backend
}

// SetBackend sets the backend instance for this session.
func (s *Session) SetBackend(b backend.Backend) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backend = b
}

// GetAppVariableID returns the root variable ID.
func (s *Session) GetAppVariableID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AppVariableID
}

// SetAppVariableID sets the root variable ID.
func (s *Session) SetAppVariableID(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AppVariableID = id
}

// AddConnection registers a new connection to this session.
func (s *Session) AddConnection(connectionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connections[connectionID] = struct{}{}
	s.lastActivity = time.Now()
}

// RemoveConnection unregisters a connection from this session.
// Calls backend.UnwatchAll to clean up watches for this connection.
// Returns true if this was the last frontend connection.
// CRC: crc-Session.md
func (s *Session) RemoveConnection(connectionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.connections, connectionID)
	s.lastActivity = time.Now()

	// Clean up watches for this connection via backend
	if s.backend != nil {
		s.backend.UnwatchAll(connectionID)
	}

	return len(s.connections) == 0
}

// IsActive checks if the session has any connections.
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections) > 0
}

// GetConnectionCount returns the number of active connections.
func (s *Session) GetConnectionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

// Touch updates the lastActivity timestamp.
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastActivity = time.Now()
}

// GetCreatedAt returns the session creation time.
func (s *Session) GetCreatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.createdAt
}

// GetLastActivity returns the last activity time.
func (s *Session) GetLastActivity() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastActivity
}

// GetConnections returns a copy of the connection IDs.
func (s *Session) GetConnections() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conns := make([]string, 0, len(s.connections))
	for id := range s.connections {
		conns = append(conns, id)
	}
	return conns
}

// GenerateSessionID creates a unique session identifier.
func GenerateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
