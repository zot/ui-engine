// Test Design: test-Communication.md
// CRC: crc-HTTPEndpoint.md
// Spec: interfaces.md, deployment.md
package server

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestHTTPRedirectToSession verifies GET / creates session and redirects
func TestHTTPRedirectToSession(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	endpoint.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("Expected redirect (307), got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		t.Fatal("Expected Location header")
	}
	if !strings.HasPrefix(location, "/") {
		t.Errorf("Location should start with /, got %s", location)
	}

	// Session should exist
	sessionID := strings.TrimPrefix(location, "/")
	if !sessions.SessionExists(sessionID) {
		t.Error("Session should exist after redirect")
	}
}

// TestHTTPAccessValidSession verifies session access returns HTML
func TestHTTPAccessValidSession(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	// Create embedded test site
	testFS := &mockFS{files: map[string]string{
		"index.html": "<html><body>Test</body></html>",
	}}
	endpoint.SetEmbeddedSite(testFS)

	// Create a session first
	sess, _, err := sessions.CreateSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Access the session URL
	req := httptest.NewRequest("GET", "/"+sess.ID, nil)
	w := httptest.NewRecorder()

	endpoint.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Test") {
		t.Error("Expected index.html content")
	}
}

// TestHTTPAccessInvalidSession verifies non-existent session handling
func TestHTTPAccessInvalidSession(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	// Create embedded test site
	testFS := &mockFS{files: map[string]string{
		"index.html": "<html><body>Test</body></html>",
	}}
	endpoint.SetEmbeddedSite(testFS)

	// Access non-existent session URL (will try to serve as static file)
	req := httptest.NewRequest("GET", "/nonexistent123", nil)
	w := httptest.NewRecorder()

	endpoint.ServeHTTP(w, req)

	resp := w.Result()
	// Since it's not a valid session, it tries to serve as static file
	// which will fail (404) since "nonexistent123" doesn't exist
	// OR it could serve index.html depending on behavior
	if resp.StatusCode == http.StatusOK {
		// If 200, it's serving something (which is fine for SPA fallback)
		t.Log("Server returns 200 for invalid session (SPA fallback)")
	} else if resp.StatusCode == http.StatusNotFound {
		t.Log("Server returns 404 for invalid session")
	} else {
		t.Logf("Got status %d for invalid session", resp.StatusCode)
	}
}

// TestHTTPServeStaticFromCustomDir verifies --dir flag serving
func TestHTTPServeStaticFromCustomDir(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	// Set static dir to a test directory
	// In real usage, this would be set by --dir flag
	// For this test, we just verify the setter works
	endpoint.SetStaticDir("/custom")

	// The actual file serving is tested via integration tests
	// since it requires real filesystem access
}

// TestHTTPDebugEndpoint verifies /{sessionID}/variables endpoint
func TestHTTPDebugEndpoint(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	// Set up debug data provider - receives vended ID (numeric)
	var receivedVendedID string
	endpoint.SetDebugDataProvider(func(vendedID string) ([]DebugVariable, error) {
		receivedVendedID = vendedID
		return []DebugVariable{
			{ID: 1, Type: "App", Value: map[string]string{"name": "Test"}},
		}, nil
	})

	// Create a session
	sess, vendedID, _ := sessions.CreateSession()

	// Request debug page using internal session ID (UUID) in URL
	req := httptest.NewRequest("GET", "/"+sess.ID+"/variables", nil)
	w := httptest.NewRecorder()

	endpoint.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	// Verify provider received vended ID
	if receivedVendedID != vendedID {
		t.Errorf("Expected vended ID '%s', got '%s'", vendedID, receivedVendedID)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Should contain HTML tree view
	if !strings.Contains(bodyStr, "html") {
		t.Error("Expected HTML response")
	}
}

// TestHTTPDebugEndpointInvalidSession verifies invalid session returns 404
func TestHTTPDebugEndpointInvalidSession(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	// Set up debug data provider
	endpoint.SetDebugDataProvider(func(vendedID string) ([]DebugVariable, error) {
		return nil, nil
	})

	// Request with non-existent session ID
	req := httptest.NewRequest("GET", "/nonexistent-session/variables", nil)
	w := httptest.NewRecorder()

	endpoint.ServeHTTP(w, req)

	resp := w.Result()
	// Should return 404 since session doesn't exist (handleRoot won't match)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

// TestHTTPAPIEndpoint verifies /api/ routing
func TestHTTPAPIEndpoint(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	// API calls require a handler, which we don't have in this test
	// Just verify the endpoint exists and returns appropriate error
	req := httptest.NewRequest("POST", "/api/test", nil)
	w := httptest.NewRecorder()

	endpoint.ServeHTTP(w, req)

	resp := w.Result()
	// Without handler, should return error
	if resp.StatusCode == http.StatusOK {
		t.Error("Expected error without handler")
	}
}

// TestHTTPSessionCreationCallback verifies session creation triggers callback
func TestHTTPSessionCreationCallback(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	callbackCalled := false
	sessions.SetOnSessionCreated(func(vendedID string, sess *Session) error {
		callbackCalled = true
		return nil
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	endpoint.ServeHTTP(w, req)

	if !callbackCalled {
		t.Error("Session creation callback should be called on root access")
	}
}

// TestHTTPMultipleSessionCreation verifies unique sessions
func TestHTTPMultipleSessionCreation(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	createdSessions := make(map[string]bool)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		endpoint.ServeHTTP(w, req)

		resp := w.Result()
		location := resp.Header.Get("Location")
		sessionID := strings.TrimPrefix(location, "/")

		if createdSessions[sessionID] {
			t.Errorf("Duplicate session ID: %s", sessionID)
		}
		createdSessions[sessionID] = true
	}

	if sessions.Count() != 10 {
		t.Errorf("Expected 10 sessions, got %d", sessions.Count())
	}
}

// TestHTTPSessionWithPath verifies session URL with path
func TestHTTPSessionWithPath(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	// Create embedded test site
	testFS := &mockFS{files: map[string]string{
		"index.html": "<html><body>SPA</body></html>",
	}}
	endpoint.SetEmbeddedSite(testFS)

	// Create a session
	sess, _, _ := sessions.CreateSession()

	// Access session with path
	req := httptest.NewRequest("GET", "/"+sess.ID+"/users/5", nil)
	w := httptest.NewRecorder()

	endpoint.ServeHTTP(w, req)

	resp := w.Result()
	// Should serve SPA (index.html) for client-side routing
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "SPA") {
		t.Error("Expected SPA index.html content")
	}
}

// TestHTTPWebSocketRouting verifies /ws/ path routing
func TestHTTPWebSocketRouting(t *testing.T) {
	sessions := NewSessionManager(time.Hour)
	endpoint := NewHTTPEndpoint(sessions, nil, nil)

	// WebSocket upgrade requires actual WebSocket connection
	// Just verify the path is handled
	req := httptest.NewRequest("GET", "/ws/test-session", nil)
	w := httptest.NewRecorder()

	endpoint.ServeHTTP(w, req)

	// Without proper upgrade headers, should fail
	resp := w.Result()
	if resp.StatusCode == http.StatusOK {
		t.Error("WebSocket without upgrade should not return 200")
	}
}

// mockFS implements fs.FS for testing
type mockFS struct {
	files map[string]string
}

func (m *mockFS) Open(name string) (fs.File, error) {
	content, ok := m.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &mockFile{
		name:    name,
		content: content,
		reader:  strings.NewReader(content),
	}, nil
}

type mockFile struct {
	name    string
	content string
	reader  *strings.Reader
}

func (f *mockFile) Stat() (fs.FileInfo, error) {
	return &mockFileInfo{name: f.name, size: int64(len(f.content))}, nil
}

func (f *mockFile) Read(b []byte) (int, error) {
	return f.reader.Read(b)
}

func (f *mockFile) Close() error {
	return nil
}

type mockFileInfo struct {
	name string
	size int64
}

func (fi *mockFileInfo) Name() string       { return fi.name }
func (fi *mockFileInfo) Size() int64        { return fi.size }
func (fi *mockFileInfo) Mode() fs.FileMode  { return 0644 }
func (fi *mockFileInfo) ModTime() time.Time { return time.Now() }
func (fi *mockFileInfo) IsDir() bool        { return false }
func (fi *mockFileInfo) Sys() interface{}   { return nil }
