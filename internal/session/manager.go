// CRC: crc-SessionManager.md
// Spec: interfaces.md, protocol.md
package session

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

// Manager manages all sessions.
type Manager struct {
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

// NewManager creates a new session manager.
func NewManager(sessionTimeout time.Duration) *Manager {
	return &Manager{
		sessions:         make(map[string]*Session),
		urlPaths:         make(map[string]map[string]int64),
		sessionTimeout:   sessionTimeout,
		nextVendedID:     1, // Vended IDs start at 1
		internalToVended: make(map[string]string),
		vendedToInternal: make(map[string]string),
	}
}

// SetOnSessionCreated sets a callback called when a session is created.
func (m *Manager) SetOnSessionCreated(callback SessionCreatedCallback) {
	m.onSessionCreated = callback
}

// SetOnSessionDestroyed sets a callback called when a session is destroyed.
func (m *Manager) SetOnSessionDestroyed(callback SessionDestroyedCallback) {
	m.onSessionDestroyed = callback
}

// CreateSession generates a new session ID and initializes the session.
// Returns the session and its vended ID (compact integer string for backend communication).
// Note: Variable 1 (app variable) is NOT created here. It's created by:
// - Lua main.lua calling session:createAppVariable() (Lua-only mode)
// - External backend via protocol (backend-only mode)
func (m *Manager) CreateSession() (*Session, string, error) {
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
func (m *Manager) GetVendedID(internalID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.internalToVended[internalID]
}

// GetInternalID returns the internal session ID for a vended ID.
// Returns empty string if session not found.
func (m *Manager) GetInternalID(vendedID string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.vendedToInternal[vendedID]
}

// GetSession retrieves a session by ID.
func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

// Get retrieves a session by ID. Returns nil if not found.
func (m *Manager) Get(id string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

// DestroySession cleans up a session and all its resources.
func (m *Manager) DestroySession(id string) error {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return nil
	}

	// Get vended ID before cleanup
	vendedID := m.internalToVended[id]

	delete(m.sessions, id)
	delete(m.urlPaths, id)
	delete(m.internalToVended, id)
	if vendedID != "" {
		delete(m.vendedToInternal, vendedID)
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
func (m *Manager) SessionExists(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.sessions[id]
	return ok
}

// RegisterURLPath associates a URL path with a presenter variable for a session.
func (m *Manager) RegisterURLPath(sessionID, path string, variableID int64) error {
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
func (m *Manager) ResolveURLPath(sessionID, path string) (int64, bool) {
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
func (m *Manager) GetAllSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// CleanupInactiveSessions removes sessions with no activity past the timeout.
func (m *Manager) CleanupInactiveSessions() int {
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
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
