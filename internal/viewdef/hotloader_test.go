// Test Design: test-HotLoader.md (Viewdef HotLoader Tests section)
package viewdef

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/zot/ui-engine/internal/config"
)

// mockSessionPusher implements SessionPusher for testing
type mockSessionPusher struct {
	sessionIDs  []string
	pushedTo    []string
	pushedDefs  map[string]map[string]string // sessionID -> key -> content
	mu          sync.Mutex
}

func newMockSessionPusher(sessionIDs []string) *mockSessionPusher {
	return &mockSessionPusher{
		sessionIDs: sessionIDs,
		pushedDefs: make(map[string]map[string]string),
	}
}

func (m *mockSessionPusher) GetSessionIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessionIDs
}

func (m *mockSessionPusher) PushViewdefs(sessionID string, viewdefs map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pushedTo = append(m.pushedTo, sessionID)
	if m.pushedDefs[sessionID] == nil {
		m.pushedDefs[sessionID] = make(map[string]string)
	}
	for k, v := range viewdefs {
		m.pushedDefs[sessionID][k] = v
	}
}

func (m *mockSessionPusher) getPushedTo() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.pushedTo))
	copy(result, m.pushedTo)
	return result
}

// Helper to create temp viewdef directory
func createTempViewdefDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "viewdef-hotloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

// Helper to create a test config
func testViewdefConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Logging.Verbosity = 0 // Quiet for tests
	return cfg
}

// === Initialization Tests ===

func TestNewViewdefHotLoader(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher(nil)

	h, err := NewHotLoader(cfg, viewdefDir, manager, sessions)
	if err != nil {
		t.Fatalf("NewHotLoader failed: %v", err)
	}
	defer h.Stop()

	if h.viewdefDir != viewdefDir {
		t.Errorf("viewdefDir = %q, want %q", h.viewdefDir, viewdefDir)
	}
	if h.watcher == nil {
		t.Error("watcher is nil")
	}
	if h.manager == nil {
		t.Error("manager is nil")
	}
	if h.sessions == nil {
		t.Error("sessions is nil")
	}
}

func TestViewdefHotLoaderStart(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher(nil)

	h, err := NewHotLoader(cfg, viewdefDir, manager, sessions)
	if err != nil {
		t.Fatalf("NewHotLoader failed: %v", err)
	}

	err = h.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer h.Stop()

	// Verify viewdef directory is being watched
	if h.watchedDirs[viewdefDir] != 1 {
		t.Errorf("viewdefDir not in watchedDirs, got %v", h.watchedDirs)
	}
}

// === File Watching Tests ===

func TestDetectHtmlFileModification(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	// Create initial html file
	htmlFile := filepath.Join(viewdefDir, "Contact.html")
	if err := os.WriteFile(htmlFile, []byte("<div>initial</div>"), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher([]string{"session1"})
	
	// Mark the viewdef as sent to the session so it receives updates
	manager.MarkViewdefSent("session1", "Contact")

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()
	defer h.Stop()

	// Wait for watcher to be ready
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(htmlFile, []byte("<div>modified</div>"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Wait for debounce (100ms) + processing
	time.Sleep(300 * time.Millisecond)

	pushedTo := sessions.getPushedTo()
	if len(pushedTo) == 0 {
		t.Error("Expected push to be triggered")
	}
}

func TestIgnoreNonHtmlFiles(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher([]string{"session1"})

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()
	defer h.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create non-html files
	cssFile := filepath.Join(viewdefDir, "styles.css")
	os.WriteFile(cssFile, []byte("body {}"), 0644)

	jsonFile := filepath.Join(viewdefDir, "data.json")
	os.WriteFile(jsonFile, []byte("{}"), 0644)

	time.Sleep(300 * time.Millisecond)

	pushedTo := sessions.getPushedTo()
	if len(pushedTo) != 0 {
		t.Errorf("Expected no pushes for non-html files, got %d", len(pushedTo))
	}
}

// === Symlink Handling Tests ===

func TestViewdefScanExistingSymlinks(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	// Create target directory and file
	targetDir := createTempViewdefDir(t)
	defer os.RemoveAll(targetDir)
	targetFile := filepath.Join(targetDir, "Contact.html")
	os.WriteFile(targetFile, []byte("<div>target</div>"), 0644)

	// Create symlink in viewdef dir
	symlinkPath := filepath.Join(viewdefDir, "Contact.html")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skipf("Cannot create symlinks: %v", err)
	}

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher(nil)

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()
	defer h.Stop()

	// Check symlink was detected
	h.mu.Lock()
	targetTracked := h.symlinkTargets[symlinkPath]
	watchCount := h.watchedDirs[targetDir]
	h.mu.Unlock()

	if targetTracked != targetDir {
		t.Errorf("symlinkTargets[%s] = %q, want %q", symlinkPath, targetTracked, targetDir)
	}
	if watchCount < 1 {
		t.Errorf("targetDir watch count = %d, want >= 1", watchCount)
	}
}

func TestViewdefReferenceCountingForSharedTargets(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	// Create shared target directory
	targetDir := createTempViewdefDir(t)
	defer os.RemoveAll(targetDir)
	os.WriteFile(filepath.Join(targetDir, "A.html"), []byte("<div>A</div>"), 0644)
	os.WriteFile(filepath.Join(targetDir, "B.html"), []byte("<div>B</div>"), 0644)

	// Create two symlinks to same directory
	symlinkA := filepath.Join(viewdefDir, "A.html")
	symlinkB := filepath.Join(viewdefDir, "B.html")
	if err := os.Symlink(filepath.Join(targetDir, "A.html"), symlinkA); err != nil {
		t.Skipf("Cannot create symlinks: %v", err)
	}
	os.Symlink(filepath.Join(targetDir, "B.html"), symlinkB)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher(nil)

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()
	defer h.Stop()

	// Both should point to same target dir
	h.mu.Lock()
	watchCount := h.watchedDirs[targetDir]
	h.mu.Unlock()

	if watchCount != 2 {
		t.Errorf("Watch count for shared target = %d, want 2", watchCount)
	}

	// Remove one symlink
	os.Remove(symlinkA)
	time.Sleep(100 * time.Millisecond)

	h.mu.Lock()
	watchCountAfter := h.watchedDirs[targetDir]
	h.mu.Unlock()

	if watchCountAfter != 1 {
		t.Errorf("Watch count after remove = %d, want 1", watchCountAfter)
	}
}

// === Debouncing Tests ===

func TestViewdefDebounceRapidChanges(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	htmlFile := filepath.Join(viewdefDir, "Contact.html")
	os.WriteFile(htmlFile, []byte("<div>v0</div>"), 0644)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher([]string{"session1"})
	manager.MarkViewdefSent("session1", "Contact")

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()
	defer h.Stop()

	time.Sleep(100 * time.Millisecond)

	// Rapid modifications
	for i := 1; i <= 5; i++ {
		os.WriteFile(htmlFile, []byte("<div>v"+string(rune('0'+i))+"</div>"), 0644)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce (100ms) + processing
	time.Sleep(300 * time.Millisecond)

	// Should only have one push due to debouncing
	pushedTo := sessions.getPushedTo()
	if len(pushedTo) != 1 {
		t.Errorf("Expected 1 push due to debouncing, got %d", len(pushedTo))
	}
}

// === Session Push Tests ===

func TestPushOnlyToAffectedSessions(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	htmlFile := filepath.Join(viewdefDir, "Contact.html")
	os.WriteFile(htmlFile, []byte("<div>initial</div>"), 0644)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	// 3 sessions, but only 2 have received the viewdef
	sessions := newMockSessionPusher([]string{"session1", "session2", "session3"})
	manager.MarkViewdefSent("session1", "Contact")
	manager.MarkViewdefSent("session2", "Contact")
	// session3 has NOT received Contact viewdef

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()
	defer h.Stop()

	time.Sleep(100 * time.Millisecond)

	// Modify the file
	os.WriteFile(htmlFile, []byte("<div>modified</div>"), 0644)
	time.Sleep(300 * time.Millisecond)

	pushedTo := sessions.getPushedTo()
	
	// Should push to session1 and session2, but not session3
	hasSession1 := false
	hasSession2 := false
	hasSession3 := false
	for _, s := range pushedTo {
		switch s {
		case "session1":
			hasSession1 = true
		case "session2":
			hasSession2 = true
		case "session3":
			hasSession3 = true
		}
	}

	if !hasSession1 {
		t.Error("Expected push to session1")
	}
	if !hasSession2 {
		t.Error("Expected push to session2")
	}
	if hasSession3 {
		t.Error("Should NOT push to session3 (hasn't received viewdef)")
	}
}

// === Graceful Shutdown Tests ===

func TestViewdefGracefulShutdown(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher(nil)

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()

	// Stop should not block or panic
	done := make(chan struct{})
	go func() {
		h.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good
	case <-time.After(time.Second):
		t.Error("Stop timed out")
	}
}

func TestViewdefShutdownWithPendingReloads(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	htmlFile := filepath.Join(viewdefDir, "Contact.html")
	os.WriteFile(htmlFile, []byte("<div>initial</div>"), 0644)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher(nil)

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()

	time.Sleep(50 * time.Millisecond)

	// Queue a change
	os.WriteFile(htmlFile, []byte("<div>modified</div>"), 0644)

	// Immediately stop (before debounce fires)
	time.Sleep(20 * time.Millisecond)
	err := h.Stop()
	if err != nil {
		t.Errorf("Stop returned error: %v", err)
	}
}

// === Edge Case Tests ===

func TestViewdefHandleDeletedFile(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	htmlFile := filepath.Join(viewdefDir, "Contact.html")
	os.WriteFile(htmlFile, []byte("<div>initial</div>"), 0644)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher(nil)

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()
	defer h.Stop()

	time.Sleep(50 * time.Millisecond)

	// Modify then delete before debounce
	os.WriteFile(htmlFile, []byte("<div>modified</div>"), 0644)
	time.Sleep(20 * time.Millisecond)
	os.Remove(htmlFile)

	// Wait for processing - should not panic
	time.Sleep(200 * time.Millisecond)
}

func TestViewdefResolveReloadPathDirect(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	htmlFile := filepath.Join(viewdefDir, "Contact.html")
	os.WriteFile(htmlFile, []byte("<div>test</div>"), 0644)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher(nil)

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()
	defer h.Stop()

	path := h.resolveReloadPath(htmlFile)
	if path != htmlFile {
		t.Errorf("resolveReloadPath(%q) = %q, want %q", htmlFile, path, htmlFile)
	}
}

func TestViewdefResolveReloadPathSymlinkTarget(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	targetDir := createTempViewdefDir(t)
	defer os.RemoveAll(targetDir)

	targetFile := filepath.Join(targetDir, "Contact.html")
	os.WriteFile(targetFile, []byte("<div>target</div>"), 0644)

	symlinkPath := filepath.Join(viewdefDir, "Contact.html")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skipf("Cannot create symlinks: %v", err)
	}

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher(nil)

	h, _ := NewHotLoader(cfg, viewdefDir, manager, sessions)
	h.Start()
	defer h.Stop()

	// Change in target dir should resolve to symlink path
	path := h.resolveReloadPath(targetFile)
	if path != symlinkPath {
		t.Errorf("resolveReloadPath(%q) = %q, want %q", targetFile, path, symlinkPath)
	}
}
