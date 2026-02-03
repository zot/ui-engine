// CRC: crc-SessionManager.md
// Spec: interfaces.md, protocol.md
package server

import (
	"strconv"
	"sync"
	"time"
)

// SessionCreatedCallback is called when a new session is created.
// Receives the vended session ID (compact integer string) and the session object.
type SessionCreatedCallback func(vendedID string, session *Session) error

// SessionDestroyedCallback is called when a session is destroyed.
// Receives the vended session ID (compact integer string) and the session object.
type SessionDestroyedCallback func(vendedID string, session *Session)

// SessionManager manages all sessions.
type SessionManager struct {
	sessions           map[string]*Session
	urlPaths           map[string]map[string]int64 // sessionID -> path -> variableID
	sessionTimeout     time.Duration
	onSessionCreated   SessionCreatedCallback
	onSessionDestroyed SessionDestroyedCallback
	mu                 sync.RWMutex

	// Vended ID mapping for backend communication
	nextVendedID     int64             // Counter for sequential vended IDs (starts at 1)
	internalToVended map[string]string // internal session ID (UUID) -> vended ID (string integer)
	vendedToInternal map[string]string // vended ID -> internal session ID
}

// NewSessionManager creates a new session manager.
func NewSessionManager(sessionTimeout time.Duration) *SessionManager {
	return &SessionManager{
		sessions:         make(map[string]*Session),
		urlPaths:         make(map[string]map[string]int64),
		sessionTimeout:   sessionTimeout,
		nextVendedID:     1, // Vended IDs start at 1
		internalToVended: make(map[string]string),
		vendedToInternal: make(map[string]string),
	}
}

// SetOnSessionCreated sets a callback called when a session is created.
func (m *SessionManager) SetOnSessionCreated(callback SessionCreatedCallback) {
	m.onSessionCreated = callback
}

// SetOnSessionDestroyed sets a callback called when a session is destroyed.
func (m *SessionManager) SetOnSessionDestroyed(callback SessionDestroyedCallback) {
	m.onSessionDestroyed = callback
}

// CreateSession generates a new session ID and initializes the session.
// Returns the session and its vended ID (compact integer string for backend communication).
// Note: Variable 1 (app variable) is NOT created here. It's created by:
// - Lua main.lua calling session:createAppVariable() (Lua-only mode)
// - External backend via protocol (backend-only mode)
func (m *SessionManager) CreateSession() (*Session, string, error) {
	internalID := GenerateSessionID()

	session := NewSession(internalID)

	m.mu.Lock()
	// Assign vended ID
	vendedID := strconv.FormatInt(m.nextVendedID, 10)
	m.nextVendedID++
	m.internalToVended[internalID] = vendedID
	m.vendedToInternal[vendedID] = internalID

	m.sessions[internalID] = session
	m.urlPaths[internalID] = make(map[string]int64)
	m.mu.Unlock()

	// Call callback to create Lua session (if enabled)
	// Pass vended ID and session for backend creation
	if m.onSessionCreated != nil {
		if err := m.onSessionCreated(vendedID, session); err != nil {
			// Session creation callback failed - clean up
			m.mu.Lock()
			delete(m.sessions, internalID)
			delete(m.urlPaths, internalID)
			delete(m.internalToVended, internalID)
			delete(m.vendedToInternal, vendedID)
			m.mu.Unlock()
			return nil, "", err
		}
	}

	return session, vendedID, nil
}

// GetVendedID returns the vended ID for an internal session ID.
// Returns empty string if session not found.
func (m *SessionManager) GetVendedID(internalID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.internalToVended[internalID]
}

// GetInternalID returns the internal session ID for a vended ID.
// Returns empty string if session not found.
func (m *SessionManager) GetInternalID(vendedID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.vendedToInternal[vendedID]
}

// GetSession retrieves a session by ID.
func (m *SessionManager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

// Get retrieves a session by ID. Returns nil if not found.
func (m *SessionManager) Get(id string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

// DestroySession cleans up a session and all its resources.
func (m *SessionManager) DestroySession(id string) error {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return nil
	}

	// Get vended ID before cleanup
	vendedID := m.internalToVended[id]

	// Clear the session's batcher if present
	if session.batcher != nil {
		session.batcher.Clear()
	}

	delete(m.sessions, id)
	delete(m.urlPaths, id)
	delete(m.internalToVended, id)
	if vendedID != "" {
		delete(m.vendedToInternal, vendedID)
	}

	// Reset vended ID counter when all sessions are destroyed
	// This ensures session 1 is always created fresh after cleanup
	if len(m.sessions) == 0 {
		m.nextVendedID = 1
	}
	m.mu.Unlock()

	// Call callback to destroy Lua backend (if enabled)
	// Pass vended ID and session for backend cleanup
	if m.onSessionDestroyed != nil && vendedID != "" {
		m.onSessionDestroyed(vendedID, session)
	}

	return nil
}

// SessionExists checks if a session ID is valid.
func (m *SessionManager) SessionExists(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.sessions[id]
	return ok
}

// RegisterURLPath associates a URL path with a presenter variable for a session.
func (m *SessionManager) RegisterURLPath(sessionID, path string, variableID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	paths, ok := m.urlPaths[sessionID]
	if !ok {
		return nil // Session doesn't exist
	}

	paths[path] = variableID
	return nil
}

// ResolveURLPath finds the presenter variable for a URL path in a session.
func (m *SessionManager) ResolveURLPath(sessionID, path string) (int64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	paths, ok := m.urlPaths[sessionID]
	if !ok {
		return 0, false
	}

	varID, ok := paths[path]
	return varID, ok
}

// GetAllSessions returns all sessions.
func (m *SessionManager) GetAllSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// CleanupInactiveSessions removes sessions with no activity past the timeout.
func (m *SessionManager) CleanupInactiveSessions() int {
	if m.sessionTimeout == 0 {
		return 0 // Never cleanup
	}

	m.mu.RLock()
	cutoff := time.Now().Add(-m.sessionTimeout)
	var toRemove []string

	for id, session := range m.sessions {
		if session.GetLastActivity().Before(cutoff) {
			toRemove = append(toRemove, id)
		}
	}
	m.mu.RUnlock()

	for _, id := range toRemove {
		m.DestroySession(id)
	}

	return len(toRemove)
}

// Count returns the number of sessions.
func (m *SessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
