// Test Design: test-HotLoader.md
package lua

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	lua "github.com/yuin/gopher-lua"
	"github.com/zot/ui-engine/internal/config"
)

// mockLuaSession implements a minimal LuaSession for testing
type mockLuaSession struct {
	ID          string
	loadedCode  []string
	loadCodeErr error
	mu          sync.Mutex
}

func (m *mockLuaSession) LoadCode(name, code string) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadedCode = append(m.loadedCode, name)
	return nil, m.loadCodeErr
}

func (m *mockLuaSession) getLoadedCode() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.loadedCode))
	copy(result, m.loadedCode)
	return result
}

// Helper to create temp lua directory
func createTempLuaDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "hotloader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

// Helper to create a mock config with luaDir's parent as Server.Dir
func testConfig(luaDir string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.Logging.Verbosity = 0             // Quiet for tests
	cfg.Server.Dir = filepath.Dir(luaDir) // baseDir is parent of lua/
	return cfg
}

// === Initialization Tests ===

func TestNewHotLoader(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	cfg := testConfig(luaDir)
	sessions := []*LuaSession{}
	getSessions := func() []*LuaSession { return sessions }

	h, err := NewHotLoader(cfg, luaDir, getSessions, nil)
	if err != nil {
		t.Fatalf("NewHotLoader failed: %v", err)
	}
	defer h.Stop()

	if h.luaDir != luaDir {
		t.Errorf("luaDir = %q, want %q", h.luaDir, luaDir)
	}
	if h.watcher == nil {
		t.Error("watcher is nil")
	}
	if h.getSessions == nil {
		t.Error("getSessions is nil")
	}
}

func TestHotLoaderStart(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	cfg := testConfig(luaDir)
	h, err := NewHotLoader(cfg, luaDir, func() []*LuaSession { return nil }, nil)
	if err != nil {
		t.Fatalf("NewHotLoader failed: %v", err)
	}

	err = h.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer h.Stop()

	// Verify lua directory is being watched
	if h.watchedDirs[luaDir] != 1 {
		t.Errorf("luaDir not in watchedDirs, got %v", h.watchedDirs)
	}
}

// === File Watching Tests ===

func TestDetectLuaFileModification(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	// Create initial lua file
	luaFile := filepath.Join(luaDir, "app.lua")
	if err := os.WriteFile(luaFile, []byte("-- initial"), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	var reloadCount atomic.Int32
	mock := &mockLuaSession{ID: "1"}

	cfg := testConfig(luaDir)
	h, _ := NewHotLoader(cfg, luaDir, func() []*LuaSession {
		reloadCount.Add(1)
		return []*LuaSession{{ID: mock.ID}}
	}, nil)
	h.Start()
	defer h.Stop()

	// Wait for watcher to be ready
	time.Sleep(50 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(luaFile, []byte("-- modified"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Wait for debounce + processing
	time.Sleep(200 * time.Millisecond)

	if reloadCount.Load() == 0 {
		t.Error("Expected reload to be triggered")
	}
}

func TestIgnoreNonLuaFiles(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	var reloadCount atomic.Int32

	cfg := testConfig(luaDir)
	h, _ := NewHotLoader(cfg, luaDir, func() []*LuaSession {
		reloadCount.Add(1)
		return nil
	}, nil)
	h.Start()
	defer h.Stop()

	time.Sleep(50 * time.Millisecond)

	// Create non-lua files
	txtFile := filepath.Join(luaDir, "notes.txt")
	os.WriteFile(txtFile, []byte("test"), 0644)

	jsonFile := filepath.Join(luaDir, "data.json")
	os.WriteFile(jsonFile, []byte("{}"), 0644)

	time.Sleep(200 * time.Millisecond)

	if reloadCount.Load() != 0 {
		t.Errorf("Expected no reloads for non-lua files, got %d", reloadCount.Load())
	}
}

// === Symlink Handling Tests ===

// === Debouncing Tests ===

func TestDebounceRapidChanges(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	luaFile := filepath.Join(luaDir, "app.lua")
	os.WriteFile(luaFile, []byte("-- v0"), 0644)

	var reloadCount atomic.Int32

	cfg := testConfig(luaDir)
	h, _ := NewHotLoader(cfg, luaDir, func() []*LuaSession {
		reloadCount.Add(1)
		return nil
	}, nil)
	h.Start()
	defer h.Stop()

	time.Sleep(50 * time.Millisecond)

	// Rapid modifications
	for i := 1; i <= 5; i++ {
		os.WriteFile(luaFile, []byte("-- v"+string(rune('0'+i))), 0644)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce
	time.Sleep(200 * time.Millisecond)

	// Should only have one reload due to debouncing
	count := reloadCount.Load()
	if count != 1 {
		t.Errorf("Expected 1 reload due to debouncing, got %d", count)
	}
}

func TestDebouncePerFile(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	fileA := filepath.Join(luaDir, "a.lua")
	fileB := filepath.Join(luaDir, "b.lua")
	os.WriteFile(fileA, []byte("-- a"), 0644)
	os.WriteFile(fileB, []byte("-- b"), 0644)

	var reloadCount atomic.Int32

	cfg := testConfig(luaDir)
	h, _ := NewHotLoader(cfg, luaDir, func() []*LuaSession {
		reloadCount.Add(1)
		return nil
	}, nil)
	h.Start()
	defer h.Stop()

	time.Sleep(50 * time.Millisecond)

	// Modify both files
	os.WriteFile(fileA, []byte("-- a modified"), 0644)
	time.Sleep(20 * time.Millisecond)
	os.WriteFile(fileB, []byte("-- b modified"), 0644)

	// Wait for both to debounce
	time.Sleep(200 * time.Millisecond)

	// Should have 2 reloads (one per file)
	count := reloadCount.Load()
	if count != 2 {
		t.Errorf("Expected 2 reloads (one per file), got %d", count)
	}
}

// === Session Reload Tests ===

func TestReloadWithNoSessions(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	luaFile := filepath.Join(luaDir, "app.lua")
	os.WriteFile(luaFile, []byte("-- initial"), 0644)

	cfg := testConfig(luaDir)
	h, _ := NewHotLoader(cfg, luaDir, func() []*LuaSession {
		return []*LuaSession{} // Empty
	}, nil)
	h.Start()
	defer h.Stop()

	time.Sleep(50 * time.Millisecond)

	// Should not panic with no sessions
	os.WriteFile(luaFile, []byte("-- modified"), 0644)
	time.Sleep(200 * time.Millisecond)

	// Test passes if no panic
}

// === Graceful Shutdown Tests ===

func TestGracefulShutdown(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	cfg := testConfig(luaDir)
	h, _ := NewHotLoader(cfg, luaDir, func() []*LuaSession { return nil }, nil)
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

func TestShutdownWithPendingReloads(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	luaFile := filepath.Join(luaDir, "app.lua")
	os.WriteFile(luaFile, []byte("-- initial"), 0644)

	cfg := testConfig(luaDir)
	h, _ := NewHotLoader(cfg, luaDir, func() []*LuaSession { return nil }, nil)
	h.Start()

	time.Sleep(50 * time.Millisecond)

	// Queue a change
	os.WriteFile(luaFile, []byte("-- modified"), 0644)

	// Immediately stop (before debounce fires)
	time.Sleep(20 * time.Millisecond)
	err := h.Stop()
	if err != nil {
		t.Errorf("Stop returned error: %v", err)
	}
}

// === Edge Case Tests ===

func TestHandleDeletedFile(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	luaFile := filepath.Join(luaDir, "app.lua")
	os.WriteFile(luaFile, []byte("-- initial"), 0644)

	cfg := testConfig(luaDir)
	h, _ := NewHotLoader(cfg, luaDir, func() []*LuaSession { return nil }, nil)
	h.Start()
	defer h.Stop()

	time.Sleep(50 * time.Millisecond)

	// Modify then delete before debounce
	os.WriteFile(luaFile, []byte("-- modified"), 0644)
	time.Sleep(20 * time.Millisecond)
	os.Remove(luaFile)

	// Wait for processing - should not panic
	time.Sleep(200 * time.Millisecond)
}

func TestResolveReloadPathDirect(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	luaFile := filepath.Join(luaDir, "app.lua")
	os.WriteFile(luaFile, []byte("-- test"), 0644)

	cfg := testConfig(luaDir)
	h, _ := NewHotLoader(cfg, luaDir, func() []*LuaSession { return nil }, nil)
	h.Start()
	defer h.Stop()

	path := h.resolveReloadPath(luaFile)
	if path != luaFile {
		t.Errorf("resolveReloadPath(%q) = %q, want %q", luaFile, path, luaFile)
	}
}

func TestResolveReloadPathSymlinkTarget(t *testing.T) {
	luaDir := createTempLuaDir(t)
	defer os.RemoveAll(luaDir)

	targetDir := createTempLuaDir(t)
	defer os.RemoveAll(targetDir)

	targetFile := filepath.Join(targetDir, "app.lua")
	os.WriteFile(targetFile, []byte("-- target"), 0644)

	symlinkPath := filepath.Join(luaDir, "app.lua")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skipf("Cannot create symlinks: %v", err)
	}

	cfg := testConfig(luaDir)
	h, _ := NewHotLoader(cfg, luaDir, func() []*LuaSession { return nil }, nil)
	h.Start()
	defer h.Stop()

	// Change in target dir should resolve to target path (not symlink)
	// With baseDir-relative tracking, we use the resolved target path directly
	path := h.resolveReloadPath(targetFile)
	if path != targetFile {
		t.Errorf("resolveReloadPath(%q) = %q, want %q", targetFile, path, targetFile)
	}
}

// reloadFixture is a standalone session that has loaded dir/app.lua (probe = 1), a
// recording executor that counts the work posted to it, and a HotLoader over it.
func reloadFixture(t *testing.T) (*HotLoader, *LuaSession, string, *int) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Server.Dir = dir
	path := filepath.Join(dir, "app.lua")
	if err := os.WriteFile(path, []byte("probe = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sess, err := NewRuntime(cfg, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sess.Shutdown)
	if err := sess.State.DoString("probe = 1"); err != nil {
		t.Fatal(err)
	}
	sess.State.SetField(sess.loadedModules, "app.lua", lua.LTrue)
	posts := 0
	sess.SetExecutor(func(fn func() (interface{}, error)) (interface{}, error) {
		posts++
		return fn()
	}, nil)
	h := &HotLoader{config: cfg, luaDir: dir, getSessions: func() []*LuaSession { return []*LuaSession{sess} }}
	return h, sess, path, &posts
}

// Test: hot reload posts its Lua work through the executor (R256)
func TestHotReloadPostsThroughExecutor(t *testing.T) {
	h, sess, path, posts := reloadFixture(t)
	if err := os.WriteFile(path, []byte("probe = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h.reloadFile(path)
	if *posts != 1 {
		t.Errorf("reload posted %d pieces of work to the executor, want 1", *posts)
	}
	if got := sess.State.GetGlobal("probe").String(); got != "2" {
		t.Errorf("probe = %s after reload, want 2", got)
	}
}

// Test: a file that does not compile never touches the session (R257)
func TestHotReloadCompileErrorTouchesNothing(t *testing.T) {
	h, sess, path, posts := reloadFixture(t)
	if err := os.WriteFile(path, []byte("probe = (\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h.reloadFile(path)
	if *posts != 0 {
		t.Errorf("a file that does not compile posted %d pieces of work, want 0", *posts)
	}
	if got := sess.State.GetGlobal("probe").String(); got != "1" {
		t.Errorf("probe = %s after a failed reload, want it unchanged at 1", got)
	}
}
