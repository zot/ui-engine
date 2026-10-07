// CRC: crc-AppDiscovery.md | Test: test-AppDiscovery.md
package site

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// mkSite builds a site with apps/<name>/ dirs and lua/<name> symlinks.
func mkSite(t *testing.T, apps ...string) string {
	t.Helper()
	siteDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(siteDir, "lua"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range apps {
		appDir := filepath.Join(siteDir, "apps", name)
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join("..", "apps", name), filepath.Join(siteDir, "lua", name)); err != nil {
			t.Skipf("cannot create symlinks: %v", err)
		}
	}
	return siteDir
}

func names(apps []App) []string {
	var out []string
	for _, app := range apps {
		out = append(out, app.Name)
	}
	return out
}

// Test: discovery keeps directory entries only (R121, R122)
func TestDiscoverKeepsDirectoryEntriesOnly(t *testing.T) {
	siteDir := mkSite(t, "a")
	lua := filepath.Join(siteDir, "lua")
	os.WriteFile(filepath.Join(lua, "main.lua"), []byte("-- main"), 0o644)
	os.Symlink(filepath.Join("..", "apps", "gone"), filepath.Join(lua, "gone"))
	// A plain directory entry is the shape a Windows junction presents.
	os.Mkdir(filepath.Join(lua, "plain"), 0o755)

	apps := Discover(siteDir, nil)
	if got, want := names(apps), []string{"a", "plain"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("apps = %v, want %v", got, want)
	}
	if apps[0].Dir != filepath.Join(lua, "a") {
		t.Errorf("Dir = %q, want the entry path %q", apps[0].Dir, filepath.Join(lua, "a"))
	}
}

// Test: ranking puts built-ins first, alphabetical within tiers (R123, R124)
func TestDiscoverRanksBuiltinsFirst(t *testing.T) {
	siteDir := mkSite(t, "zeta", "alpha", "mcp", "ark")
	got := names(Discover(siteDir, []string{"mcp", "ark", "missing"}))
	if want := []string{"ark", "mcp", "alpha", "zeta"}; !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// Test: viewdef sources end with the site directory, missing dirs omitted (R120, R125)
func TestViewdefSourcesOrderAndOmission(t *testing.T) {
	siteDir := mkSite(t, "a", "b")
	os.Mkdir(filepath.Join(siteDir, "apps", "a", "viewdefs"), 0o755)
	os.Mkdir(filepath.Join(siteDir, "viewdefs"), 0o755)
	apps := Discover(siteDir, nil)

	got := ViewdefSources(siteDir, apps)
	want := []Source{
		{Label: "a", Dir: filepath.Join(siteDir, "lua", "a", "viewdefs")},
		{Label: SiteSource, Dir: filepath.Join(siteDir, "viewdefs")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sources = %v, want %v", got, want)
	}

	os.Remove(filepath.Join(siteDir, "viewdefs"))
	if got := ViewdefSources(siteDir, apps); !reflect.DeepEqual(got, want[:1]) {
		t.Errorf("without site viewdefs, sources = %v, want %v", got, want[:1])
	}
}
