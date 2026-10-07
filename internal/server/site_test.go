// CRC: crc-Server.md | Test: test-AppDiscovery.md
package server

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zot/ui-engine/internal/config"
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
