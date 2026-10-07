// CRC: crc-HTTPEndpoint.md | Test: test-AppDiscovery.md
package server

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/zot/ui-engine/internal/site/sitetest"
)

// staticSite builds a site with app "a" (apps/a/css/x.css) activated through
// lua/a, and returns an endpoint serving it.
func staticSite(t *testing.T) (*HTTPEndpoint, string) {
	t.Helper()
	siteDir := t.TempDir()
	os.MkdirAll(filepath.Join(siteDir, "apps", "a", "css"), 0o755)
	os.WriteFile(filepath.Join(siteDir, "apps", "a", "css", "x.css"), []byte("app"), 0o644)
	os.WriteFile(filepath.Join(siteDir, "apps", "a", "app.lua"), []byte("-- lua"), 0o644)
	os.MkdirAll(filepath.Join(siteDir, "html"), 0o755)
	os.MkdirAll(filepath.Join(siteDir, "lua"), 0o755)
	sitetest.LinkDir(t, filepath.Join("..", "apps", "a"), filepath.Join(siteDir, "lua", "a"))
	return &HTTPEndpoint{staticDir: filepath.Join(siteDir, "html"), appsDir: filepath.Join(siteDir, "lua")}, siteDir
}

func get(h *HTTPEndpoint, path string) (int, string) {
	rec := httptest.NewRecorder()
	h.serveStatic(rec, httptest.NewRequest("GET", path, nil), path[1:])
	body, _ := io.ReadAll(rec.Body)
	return rec.Code, string(body)
}

// Test: an active app's whole directory is served at /<app>/ (R138)
func TestAppDirectoryIsServed(t *testing.T) {
	h, _ := staticSite(t)
	for path, want := range map[string]string{"/a/css/x.css": "app", "/a/app.lua": "-- lua"} {
		if code, body := get(h, path); code != 200 || body != want {
			t.Errorf("GET %s = %d %q, want 200 %q", path, code, body, want)
		}
	}
	if code, _ := get(h, "/nosuch/x.css"); code != 404 {
		t.Errorf("GET /nosuch/x.css = %d, want 404", code)
	}
}

// Test: a site html/ file overrides the app's file (R139)
func TestSiteFileOverridesAppFile(t *testing.T) {
	h, siteDir := staticSite(t)
	os.MkdirAll(filepath.Join(siteDir, "html", "a", "css"), 0o755)
	os.WriteFile(filepath.Join(siteDir, "html", "a", "css", "x.css"), []byte("site"), 0o644)
	if code, body := get(h, "/a/css/x.css"); code != 200 || body != "site" {
		t.Errorf("GET /a/css/x.css = %d %q, want 200 \"site\"", code, body)
	}
}

// Test: no app serving without a site directory's apps dir (R140)
func TestNoAppServingWithoutAppsDir(t *testing.T) {
	h, _ := staticSite(t)
	h.appsDir = ""
	if code, _ := get(h, "/a/css/x.css"); code != 404 {
		t.Errorf("GET /a/css/x.css without appsDir = %d, want 404", code)
	}
}
