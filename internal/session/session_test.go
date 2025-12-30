// Test Design: test-Session.md
// CRC: crc-Session.md, crc-SessionManager.md
// Spec: interfaces.md, main.md
package session

import (
	"sync"
	"testing"
	"time"
)

// TestCreateNewSession verifies basic session creation
func TestCreateNewSession(t *testing.T) {
	manager := NewManager(time.Hour)

	session, vendedID, err := manager.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session == nil {
		t.Fatal("Expected non-nil session")
	}

	if vendedID != "1" {
		t.Errorf("Expected first vended ID to be '1', got '%s'", vendedID)
	}

	if session.ID == "" {
		t.Error("Expected non-empty internal session ID")
	}

	// Verify session is accessible
	retrieved, ok := manager.GetSession(session.ID)
	if !ok {
		t.Error("Expected to find session by internal ID")
	}
	if retrieved != session {
		t.Error("Retrieved session should be same object")
	}

	if manager.Count() != 1 {
		t.Errorf("Expected 1 session, got %d", manager.Count())
	}
}

// TestSessionIDUniqueness verifies unique session IDs
func TestSessionIDUniqueness(t *testing.T) {
	manager := NewManager(time.Hour)
	ids := make(map[string]bool)
	vendedIDs := make(map[string]bool)

	for i := 0; i < 100; i++ {
		session, vendedID, err := manager.CreateSession()
		if err != nil {
			t.Fatalf("CreateSession %d failed: %v", i, err)
		}

		if ids[session.ID] {
			t.Errorf("Duplicate internal session ID: %s", session.ID)
		}
		ids[session.ID] = true

		if vendedIDs[vendedID] {
			t.Errorf("Duplicate vended session ID: %s", vendedID)
		}
		vendedIDs[vendedID] = true
	}

	if manager.Count() != 100 {
		t.Errorf("Expected 100 sessions, got %d", manager.Count())
	}
}

// TestAccessExistingSession verifies session lookup
func TestAccessExistingSession(t *testing.T) {
	manager := NewManager(time.Hour)

	session, _, err := manager.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Access by internal ID
	retrieved, ok := manager.GetSession(session.ID)
	if !ok {
		t.Error("Expected to find session by internal ID")
	}
	if retrieved.ID != session.ID {
		t.Error("Session ID mismatch")
	}

	// Access via Get (returns nil if not found)
	retrieved = manager.Get(session.ID)
	if retrieved == nil {
		t.Error("Expected to find session via Get")
	}
}

// TestAccessInvalidSession verifies error for non-existent session
func TestAccessInvalidSession(t *testing.T) {
	manager := NewManager(time.Hour)

	_, ok := manager.GetSession("nonexistent123")
	if ok {
		t.Error("Expected session not found")
	}

	session := manager.Get("nonexistent123")
	if session != nil {
		t.Error("Expected nil for non-existent session")
	}

	if manager.SessionExists("nonexistent123") {
		t.Error("Expected SessionExists to return false")
	}
}

// TestRegisterURLPath verifies path registration
func TestRegisterURLPath(t *testing.T) {
	manager := NewManager(time.Hour)

	session, _, err := manager.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	err = manager.RegisterURLPath(session.ID, "/users", 42)
	if err != nil {
		t.Fatalf("RegisterURLPath failed: %v", err)
	}

	varID, ok := manager.ResolveURLPath(session.ID, "/users")
	if !ok {
		t.Error("Expected to find registered path")
	}
	if varID != 42 {
		t.Errorf("Expected variable ID 42, got %d", varID)
	}
}

// TestURLPathResolution verifies registered path lookup
func TestURLPathResolution(t *testing.T) {
	manager := NewManager(time.Hour)

	session, _, err := manager.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Register path
	manager.RegisterURLPath(session.ID, "/users", 10)

	// Resolve registered path
	varID, ok := manager.ResolveURLPath(session.ID, "/users")
	if !ok || varID != 10 {
		t.Errorf("Expected varID 10, got %d (found: %v)", varID, ok)
	}

	// Resolve unregistered path
	_, ok = manager.ResolveURLPath(session.ID, "/other")
	if ok {
		t.Error("Expected unregistered path to return false")
	}

	// Resolve for non-existent session
	_, ok = manager.ResolveURLPath("nonexistent", "/users")
	if ok {
		t.Error("Expected non-existent session to return false")
	}
}

// TestSessionConnectionTracking verifies connection add/remove
func TestSessionConnectionTracking(t *testing.T) {
	session := NewSession("test-session")

	if session.IsActive() {
		t.Error("New session should not be active")
	}
	if session.GetConnectionCount() != 0 {
		t.Error("New session should have 0 connections")
	}

	// Add frontend connection
	session.AddConnection("frontend-1")
	if !session.IsActive() {
		t.Error("Session should be active with connection")
	}
	if session.GetConnectionCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", session.GetConnectionCount())
	}

	// Add backend connection
	session.AddConnection("backend-1")
	if session.GetConnectionCount() != 2 {
		t.Errorf("Expected 2 connections, got %d", session.GetConnectionCount())
	}

	// Remove frontend connection
	wasLast := session.RemoveConnection("frontend-1")
	if wasLast {
		t.Error("Should not be last connection")
	}
	if session.GetConnectionCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", session.GetConnectionCount())
	}

	// Remove backend connection (last)
	wasLast = session.RemoveConnection("backend-1")
	if !wasLast {
		t.Error("Should be last connection")
	}
	if session.IsActive() {
		t.Error("Session should not be active with no connections")
	}
}

// TestSessionCleanupOnInactivity verifies inactive session cleanup
func TestSessionCleanupOnInactivity(t *testing.T) {
	// Very short timeout for testing
	manager := NewManager(10 * time.Millisecond)

	session, _, err := manager.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Session exists initially
	if !manager.SessionExists(session.ID) {
		t.Error("Session should exist initially")
	}

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// Cleanup should remove the session
	removed := manager.CleanupInactiveSessions()
	if removed != 1 {
		t.Errorf("Expected 1 session removed, got %d", removed)
	}

	// Session should be gone
	if manager.SessionExists(session.ID) {
		t.Error("Session should be cleaned up")
	}
}

// TestSessionDestroyCleanup verifies session destruction
func TestSessionDestroyCleanup(t *testing.T) {
	manager := NewManager(time.Hour)

	session, vendedID, err := manager.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Verify mappings exist
	if manager.GetVendedID(session.ID) != vendedID {
		t.Error("Vended ID mapping should exist")
	}
	if manager.GetInternalID(vendedID) != session.ID {
		t.Error("Internal ID mapping should exist")
	}

	// Destroy session
	err = manager.DestroySession(session.ID)
	if err != nil {
		t.Fatalf("DestroySession failed: %v", err)
	}

	// Verify cleanup
	if manager.SessionExists(session.ID) {
		t.Error("Session should not exist after destroy")
	}
	if manager.GetVendedID(session.ID) != "" {
		t.Error("Vended ID mapping should be removed")
	}
	if manager.GetInternalID(vendedID) != "" {
		t.Error("Internal ID mapping should be removed")
	}
}

// TestVendedIDMapping verifies internal <-> vended ID mapping
func TestVendedIDMapping(t *testing.T) {
	manager := NewManager(time.Hour)

	session1, vendedID1, _ := manager.CreateSession()
	session2, vendedID2, _ := manager.CreateSession()

	// Verify sequential vended IDs
	if vendedID1 != "1" || vendedID2 != "2" {
		t.Errorf("Expected vended IDs '1' and '2', got '%s' and '%s'", vendedID1, vendedID2)
	}

	// Verify bidirectional mapping
	if manager.GetVendedID(session1.ID) != vendedID1 {
		t.Error("GetVendedID failed for session1")
	}
	if manager.GetInternalID(vendedID2) != session2.ID {
		t.Error("GetInternalID failed for session2")
	}

	// Non-existent mappings
	if manager.GetVendedID("nonexistent") != "" {
		t.Error("Expected empty string for non-existent internal ID")
	}
	if manager.GetInternalID("999") != "" {
		t.Error("Expected empty string for non-existent vended ID")
	}
}

// TestSessionTouch verifies activity timestamp update
func TestSessionTouch(t *testing.T) {
	session := NewSession("test")

	initial := session.GetLastActivity()
	time.Sleep(5 * time.Millisecond)

	session.Touch()
	updated := session.GetLastActivity()

	if !updated.After(initial) {
		t.Error("Touch should update lastActivity timestamp")
	}
}

// TestSessionCallbacks verifies create/destroy callbacks
func TestSessionCallbacks(t *testing.T) {
	manager := NewManager(time.Hour)

	var createdVendedID string
	var destroyedVendedID string

	manager.SetOnSessionCreated(func(vendedID string, session *Session) error {
		createdVendedID = vendedID
		return nil
	})

	manager.SetOnSessionDestroyed(func(vendedID string, session *Session) {
		destroyedVendedID = vendedID
	})

	session, vendedID, err := manager.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if createdVendedID != vendedID {
		t.Errorf("OnSessionCreated callback received wrong ID: %s (expected %s)", createdVendedID, vendedID)
	}

	manager.DestroySession(session.ID)

	if destroyedVendedID != vendedID {
		t.Errorf("OnSessionDestroyed callback received wrong ID: %s (expected %s)", destroyedVendedID, vendedID)
	}
}

// TestGetConnections verifies connection list retrieval
func TestGetConnections(t *testing.T) {
	session := NewSession("test")

	session.AddConnection("conn-1")
	session.AddConnection("conn-2")

	conns := session.GetConnections()
	if len(conns) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(conns))
	}

	// Verify both connections are present
	connMap := make(map[string]bool)
	for _, c := range conns {
		connMap[c] = true
	}
	if !connMap["conn-1"] || !connMap["conn-2"] {
		t.Error("Missing expected connection IDs")
	}
}

// TestGenerateSessionID verifies ID generation properties
func TestGenerateSessionID(t *testing.T) {
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := GenerateSessionID()

		// Should be non-empty
		if id == "" {
			t.Error("Generated ID should not be empty")
		}

		// Should be URL-safe (hex encoded)
		if len(id) != 32 { // 16 bytes = 32 hex chars
			t.Errorf("Expected 32 char ID, got %d chars: %s", len(id), id)
		}

		// Should be unique
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

// TestConcurrentSessionAccess verifies thread-safety
func TestConcurrentSessionAccess(t *testing.T) {
	manager := NewManager(time.Hour)
	session, _, _ := manager.CreateSession()

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	// Concurrent connection adds/removes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			connID := "conn-" + string(rune('A'+id))
			for j := 0; j < iterations; j++ {
				session.AddConnection(connID)
				session.Touch()
				session.GetConnectionCount()
				session.IsActive()
				session.RemoveConnection(connID)
			}
		}(i)
	}

	wg.Wait()

	// Final state should be no connections
	if session.GetConnectionCount() != 0 {
		t.Errorf("Expected 0 connections after concurrent operations, got %d", session.GetConnectionCount())
	}
}

// TestConcurrentManagerAccess verifies manager thread-safety
func TestConcurrentManagerAccess(t *testing.T) {
	manager := NewManager(time.Hour)

	var wg sync.WaitGroup
	const goroutines = 10

	// Concurrent session creates
	sessions := make(chan *Session, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session, _, err := manager.CreateSession()
			if err != nil {
				t.Errorf("CreateSession failed: %v", err)
				return
			}
			sessions <- session
		}()
	}

	wg.Wait()
	close(sessions)

	if manager.Count() != goroutines {
		t.Errorf("Expected %d sessions, got %d", goroutines, manager.Count())
	}

	// Concurrent session destroys
	for session := range sessions {
		wg.Add(1)
		go func(s *Session) {
			defer wg.Done()
			manager.DestroySession(s.ID)
		}(session)
	}

	wg.Wait()

	if manager.Count() != 0 {
		t.Errorf("Expected 0 sessions after cleanup, got %d", manager.Count())
	}
}

// TestNoCleanupWithZeroTimeout verifies cleanup is disabled with 0 timeout
func TestNoCleanupWithZeroTimeout(t *testing.T) {
	manager := NewManager(0) // 0 = never cleanup

	_, _, err := manager.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Cleanup should remove nothing
	removed := manager.CleanupInactiveSessions()
	if removed != 0 {
		t.Errorf("Expected 0 removed with 0 timeout, got %d", removed)
	}

	if manager.Count() != 1 {
		t.Error("Session should still exist")
	}
}

// TestGetAllSessions verifies listing all sessions
func TestGetAllSessions(t *testing.T) {
	manager := NewManager(time.Hour)

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		_, _, err := manager.CreateSession()
		if err != nil {
			t.Fatalf("CreateSession %d failed: %v", i, err)
		}
	}

	sessions := manager.GetAllSessions()
	if len(sessions) != 5 {
		t.Errorf("Expected 5 sessions, got %d", len(sessions))
	}
}
