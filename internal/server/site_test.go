// CRC: crc-Server.md | Test: test-AppDiscovery.md
package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zot/ui-engine/internal/config"
	"github.com/zot/ui-engine/internal/site/sitetest"
	"github.com/zot/ui-engine/internal/viewdef"
)

// Test: missing site directory is a start error (R119)
func TestMissingSiteDirIsStartError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-site")
	cfg := config.DefaultConfig()
	cfg.Server.Dir = missing
	cfg.Server.Socket = filepath.Join(t.TempDir(), "ui.sock")
	cfg.Lua.Enabled = false

	s := New(cfg)
	url, err := s.StartAsync(0)
	if err == nil {
		t.Fatalf("StartAsync succeeded at %s, want an error naming %s", url, missing)
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error %q does not name %s", err, missing)
	}
}

// Test: the offline resolver agrees with a running server (R143, R144)
func TestServerViewdefCollisions(t *testing.T) {
	siteDir := t.TempDir()
	os.MkdirAll(filepath.Join(siteDir, "lua"), 0o755)
	for _, name := range []string{"aa", "zz"} {
		vd := filepath.Join(siteDir, "apps", name, "viewdefs")
		os.MkdirAll(vd, 0o755)
		os.WriteFile(filepath.Join(vd, "T.DEFAULT.html"), []byte("<template>"+name+"</template>"), 0o644)
		sitetest.LinkDir(t, filepath.Join("..", "apps", name), filepath.Join(siteDir, "lua", name))
	}
	cfg := config.DefaultConfig()
	cfg.Server.Dir = siteDir
	cfg.Server.BuiltinApps = []string{"zz"}
	cfg.Server.Socket = filepath.Join(t.TempDir(), "ui.sock")
	cfg.Lua.Enabled = false

	got := New(cfg).ViewdefCollisions()
	want := viewdef.ResolveCollisions(siteDir, cfg.Server.BuiltinApps)
	if len(got) != 1 || len(want) != 1 || got[0] != want[0] || got[0].Winner != "zz" {
		t.Errorf("server = %+v, resolver = %+v, want one zz-over-aa collision from both", got, want)
	}
}
