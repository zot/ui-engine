// CRC: crc-ViewdefStore.md | Test: test-HotLoader.md
// Viewdef HotLoader Tests section; app-discovery cases are under test-AppDiscovery.md
package viewdef

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/zot/ui-engine/internal/config"
	"github.com/zot/ui-engine/internal/site"
)

// mockSessionPusher implements SessionPusher for testing
type mockSessionPusher struct {
	sessionIDs []string
	pushedTo   []string
	pushedDefs map[string]map[string]string // sessionID -> key -> content
	mu         sync.Mutex
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

// Helper to create a temp site and return its viewdefs/ directory
func createTempViewdefDir(t *testing.T) string {
	dir := filepath.Join(t.TempDir(), "viewdefs")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("Failed to create viewdefs dir: %v", err)
	}
	return dir
}

// newTestHotLoader points cfg at viewdefDir's site, loads the manager from
// the site's sources (so MarkViewdefSent finds the viewdefs), and creates
// the hot loader.
func newTestHotLoader(cfg *config.Config, viewdefDir string, manager *ViewdefManager, sessions SessionPusher) (*HotLoader, error) {
	cfg.Server.Dir = filepath.Dir(viewdefDir)
	manager.LoadFromSources(site.ViewdefSources(cfg.Server.Dir, site.Discover(cfg.Server.Dir, nil)))
	return NewHotLoader(cfg, manager, sessions)
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

	h, err := newTestHotLoader(cfg, viewdefDir, manager, sessions)
	if err != nil {
		t.Fatalf("NewHotLoader failed: %v", err)
	}
	defer h.Stop()

	if h.luaDir != filepath.Join(filepath.Dir(viewdefDir), "lua") {
		t.Errorf("luaDir = %q, want the site's lua/", h.luaDir)
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

	h, err := newTestHotLoader(cfg, viewdefDir, manager, sessions)
	if err != nil {
		t.Fatalf("NewHotLoader failed: %v", err)
	}

	err = h.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer h.Stop()

	// Verify viewdef directory is being watched
	if !h.watchedSources[viewdefDir] {
		t.Errorf("viewdefDir not in watchedSources, got %v", h.watchedSources)
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
	h, _ := newTestHotLoader(cfg, viewdefDir, manager, sessions)
	manager.MarkViewdefSent("session1", "Contact")

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

	h, _ := newTestHotLoader(cfg, viewdefDir, manager, sessions)
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

// === Debouncing Tests ===

func TestViewdefDebounceRapidChanges(t *testing.T) {
	viewdefDir := createTempViewdefDir(t)
	defer os.RemoveAll(viewdefDir)

	htmlFile := filepath.Join(viewdefDir, "Contact.html")
	os.WriteFile(htmlFile, []byte("<div>v0</div>"), 0644)

	cfg := testViewdefConfig()
	manager := NewViewdefManager()
	sessions := newMockSessionPusher([]string{"session1"})
	h, _ := newTestHotLoader(cfg, viewdefDir, manager, sessions)
	manager.MarkViewdefSent("session1", "Contact")

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
	h, _ := newTestHotLoader(cfg, viewdefDir, manager, sessions)
	manager.MarkViewdefSent("session1", "Contact")
	manager.MarkViewdefSent("session2", "Contact")
	// session3 has NOT received Contact viewdef

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

	h, _ := newTestHotLoader(cfg, viewdefDir, manager, sessions)
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

	h, _ := newTestHotLoader(cfg, viewdefDir, manager, sessions)
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

	h, _ := newTestHotLoader(cfg, viewdefDir, manager, sessions)
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

// Test: test-AppDiscovery.md

// Test: activating and deactivating an app re-runs discovery (R129)
func TestActivationRegistersAppViewdefs(t *testing.T) {
	siteDir := t.TempDir()
	os.MkdirAll(filepath.Join(siteDir, "lua"), 0o755)
	appVD := filepath.Join(siteDir, "apps", "b", "viewdefs")
	os.MkdirAll(appVD, 0o755)
	os.WriteFile(filepath.Join(appVD, "B.DEFAULT.html"), []byte("<template>v1</template>"), 0o644)

	cfg := testViewdefConfig()
	cfg.Server.Dir = siteDir
	manager := NewViewdefManager()
	manager.LoadFromSources(site.ViewdefSources(siteDir, site.Discover(siteDir, nil)))
	sessions := newMockSessionPusher([]string{"s1"})
	h, _ := NewHotLoader(cfg, manager, sessions)
	h.Start()
	defer h.Stop()
	time.Sleep(50 * time.Millisecond)

	entry := filepath.Join(siteDir, "lua", "b")
	if err := os.Symlink(filepath.Join("..", "apps", "b"), entry); err != nil {
		t.Skipf("cannot create symlinks: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if _, ok := manager.GetViewdefsForType("B")["B.DEFAULT"]; !ok {
		t.Fatal("B.DEFAULT not registered after activation")
	}

	manager.MarkViewdefSent("s1", "B.DEFAULT")
	os.WriteFile(filepath.Join(appVD, "B.DEFAULT.html"), []byte("<template>v2</template>"), 0o644)
	time.Sleep(300 * time.Millisecond)
	if len(sessions.getPushedTo()) == 0 {
		t.Error("change in an activated app's viewdef was not pushed")
	}

	os.Remove(entry)
	time.Sleep(200 * time.Millisecond)
	if _, ok := manager.GetViewdefsForType("B")["B.DEFAULT"]; ok {
		t.Error("B.DEFAULT still registered after deactivation")
	}
}

// Test: site viewdefs watched only if present at startup (R131, R133)
func TestSiteViewdefsCreatedLaterWaitsForRestart(t *testing.T) {
	siteDir := t.TempDir()
	os.MkdirAll(filepath.Join(siteDir, "lua"), 0o755)
	cfg := testViewdefConfig()
	cfg.Server.Dir = siteDir
	manager := NewViewdefManager()
	manager.LoadFromSources(site.ViewdefSources(siteDir, nil))
	h, _ := NewHotLoader(cfg, manager, newMockSessionPusher(nil))
	h.Start()
	defer h.Stop()
	time.Sleep(50 * time.Millisecond)

	vd := filepath.Join(siteDir, "viewdefs")
	os.Mkdir(vd, 0o755)
	time.Sleep(50 * time.Millisecond)
	os.WriteFile(filepath.Join(vd, "S.DEFAULT.html"), []byte("<template/>"), 0o644)
	time.Sleep(300 * time.Millisecond)
	if manager.Count() != 0 {
		t.Errorf("site viewdefs created after start were registered (%d)", manager.Count())
	}
}
