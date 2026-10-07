// CRC: crc-LuaHotLoader.md | Test: test-AppDiscovery.md
package lua

import (
	"github.com/zot/ui-engine/internal/site/sitetest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zot/ui-engine/internal/config"
)

// appSite builds a site whose app "a" lives at appDir, reached through lua/a.
func appSite(t *testing.T, appDir string) *config.Config {
	t.Helper()
	siteDir := t.TempDir()
	os.MkdirAll(filepath.Join(siteDir, "lua"), 0o755)
	sitetest.LinkDir(t, appDir, filepath.Join(siteDir, "lua", "a"))
	cfg := config.DefaultConfig()
	cfg.Logging.Verbosity = 0
	cfg.Server.Dir = siteDir
	return cfg
}

// startCounting starts a hot loader that counts reloads (each reload asks
// for the sessions once).
func startCounting(t *testing.T, cfg *config.Config) (*HotLoader, *atomic.Int32) {
	t.Helper()
	var reloads atomic.Int32
	h, err := NewHotLoader(cfg, filepath.Join(cfg.Server.Dir, "lua"), func() []*LuaSession {
		reloads.Add(1)
		return nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.Stop() })
	time.Sleep(50 * time.Millisecond)
	return h, &reloads
}

// Test: app module outside the site hot-reloads (R132, R134)
func TestAppModuleOutsideSiteReloads(t *testing.T) {
	appDir := t.TempDir()
	mod := filepath.Join(appDir, "mod.lua")
	os.WriteFile(mod, []byte("-- v1"), 0o644)
	_, reloads := startCounting(t, appSite(t, appDir))

	os.WriteFile(mod, []byte("-- v2"), 0o644)
	time.Sleep(300 * time.Millisecond)
	if reloads.Load() == 0 {
		t.Error("module of an app outside the site was not reloaded")
	}
}

// Test: one file reached by two watched paths reloads once (R134).
// Queues directly: on Linux inotify gives both paths one watch and so one
// event, which would let a watcher-driven test pass vacuously; Windows
// watches each path separately and does deliver two.
func TestFileSeenTwiceReloadsOnce(t *testing.T) {
	siteDir := t.TempDir()
	appDir := filepath.Join(siteDir, "apps", "a")
	os.MkdirAll(appDir, 0o755)
	os.WriteFile(filepath.Join(appDir, "app.lua"), []byte("-- v1"), 0o644)
	os.MkdirAll(filepath.Join(siteDir, "lua"), 0o755)
	sitetest.LinkDir(t, filepath.Join("..", "apps", "a"), filepath.Join(siteDir, "lua", "a"))
	cfg := config.DefaultConfig()
	cfg.Logging.Verbosity = 0
	cfg.Server.Dir = siteDir
	h, err := NewHotLoader(cfg, filepath.Join(siteDir, "lua"), func() []*LuaSession { return nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Stop()

	h.queueReload(filepath.Join(appDir, "app.lua"))
	h.queueReload(filepath.Join(siteDir, "lua", "a", "app.lua"))
	if got := len(h.pendingReloads); got != 1 {
		t.Errorf("pending reloads = %d, want 1 for one file seen through two paths", got)
	}
}

// Test: activating and deactivating an app re-runs discovery (R129, R130)
func TestActivationRediscoversWatches(t *testing.T) {
	siteDir := t.TempDir()
	os.MkdirAll(filepath.Join(siteDir, "lua"), 0o755)
	appDir := t.TempDir()
	mod := filepath.Join(appDir, "mod.lua")
	os.WriteFile(mod, []byte("-- v1"), 0o644)
	cfg := config.DefaultConfig()
	cfg.Logging.Verbosity = 0
	cfg.Server.Dir = siteDir
	_, reloads := startCounting(t, cfg)

	entry := filepath.Join(siteDir, "lua", "b")
	sitetest.LinkDir(t, appDir, entry)
	time.Sleep(100 * time.Millisecond)
	os.WriteFile(mod, []byte("-- v2"), 0o644)
	time.Sleep(300 * time.Millisecond)
	if reloads.Load() == 0 {
		t.Fatal("newly activated app was not watched")
	}

	os.Remove(entry)
	time.Sleep(100 * time.Millisecond)
	before := reloads.Load()
	os.WriteFile(mod, []byte("-- v3"), 0o644)
	time.Sleep(300 * time.Millisecond)
	if reloads.Load() != before {
		t.Error("deactivated app still reloads")
	}
}
