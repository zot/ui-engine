// Package session implements session management for the UI server.
// CRC: crc-Session.md, crc-SessionManager.md
// Spec: main.md, interfaces.md
package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session represents a single user session.
type Session struct {
	ID            string
	AppVariableID int64               // Variable 1 - root app variable
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
// Returns true if this was the last frontend connection.
func (s *Session) RemoveConnection(connectionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.connections, connectionID)
	s.lastActivity = time.Now()

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
