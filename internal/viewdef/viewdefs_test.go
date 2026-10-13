// CRC: crc-ViewdefManager.md | Test: test-AppDiscovery.md
package viewdef

import (
	"fmt"
	"github.com/zot/ui-engine/internal/site/sitetest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zot/ui-engine/internal/site"
)

// twoAppSite builds a site whose apps aa and zz both ship T.DEFAULT.html.
func twoAppSite(t *testing.T) string {
	t.Helper()
	siteDir := t.TempDir()
	os.MkdirAll(filepath.Join(siteDir, "lua"), 0o755)
	for _, name := range []string{"aa", "zz"} {
		vd := filepath.Join(siteDir, "apps", name, "viewdefs")
		os.MkdirAll(vd, 0o755)
		os.WriteFile(filepath.Join(vd, "T.DEFAULT.html"), []byte("<template>"+name+"</template>"), 0o644)
		sitetest.LinkDir(t, filepath.Join("..", "apps", name), filepath.Join(siteDir, "lua", name))
	}
	return siteDir
}

func loadSite(siteDir string, builtins []string) (*ViewdefManager, *[]string) {
	var logs []string
	m := NewViewdefManager()
	m.SetLogger(func(level int, format string, args ...interface{}) {
		logs = append(logs, fmt.Sprintf(format, args...))
	})
	m.LoadFromSources(site.ViewdefSources(siteDir, site.Discover(siteDir, builtins)))
	return m, &logs
}

// Test: collision goes to the higher-ranked source and is logged (R126, R127)
func TestCollisionHigherRankWinsAndIsLogged(t *testing.T) {
	siteDir := twoAppSite(t)

	m, logs := loadSite(siteDir, nil)
	if got := m.GetViewdefsForType("T")["T.DEFAULT"]; !strings.Contains(got, "aa") {
		t.Errorf("without builtins, T.DEFAULT = %q, want aa's", got)
	}
	if len(*logs) != 1 || !strings.Contains((*logs)[0], "T.DEFAULT") ||
		!strings.Contains((*logs)[0], "aa wins over zz") {
		t.Errorf("collision log = %q, want one line naming T.DEFAULT, aa and zz", *logs)
	}

	m, _ = loadSite(siteDir, []string{"zz"})
	if got := m.GetViewdefsForType("T")["T.DEFAULT"]; !strings.Contains(got, "zz") {
		t.Errorf("with zz built-in, T.DEFAULT = %q, want zz's", got)
	}
}

// Test: on-demand loading honors the same rule (R128)
func TestOnDemandLoadingHonorsRank(t *testing.T) {
	siteDir := twoAppSite(t)
	m, _ := loadSite(siteDir, nil)
	for _, name := range []string{"aa", "zz"} {
		os.WriteFile(filepath.Join(siteDir, "apps", name, "viewdefs", "U.DEFAULT.html"),
			[]byte("<template>"+name+"</template>"), 0o644)
	}
	m.LoadViewdefsForType("U")
	if got := m.GetViewdefsForType("U")["U.DEFAULT"]; !strings.Contains(got, "aa") {
		t.Errorf("on-demand U.DEFAULT = %q, want aa's", got)
	}
}

// Test: a change to the losing copy of a colliding viewdef leaves the winner (R126)
func TestLoserChangeDoesNotDisplaceWinner(t *testing.T) {
	siteDir := twoAppSite(t)
	m, _ := loadSite(siteDir, nil)
	loser := filepath.Join(siteDir, "lua", "zz", "viewdefs", "T.DEFAULT.html")
	os.WriteFile(loser, []byte("<template>zz v2</template>"), 0o644)

	if _, _, ok := m.ApplyFileChange(loser); ok {
		t.Error("ApplyFileChange accepted the losing source's file")
	}
	if got := m.GetViewdefsForType("T")["T.DEFAULT"]; !strings.Contains(got, "aa") {
		t.Errorf("T.DEFAULT = %q after the loser changed, want aa's", got)
	}
}

// Test: one file reached through two paths is not a collision (R137)
func TestSameFileIsNotACollision(t *testing.T) {
	siteDir := twoAppSite(t)
	siteVD := filepath.Join(siteDir, "viewdefs")
	os.Mkdir(siteVD, 0o755)
	// An old-scheme per-viewdef link pointing into the app aa.
	if err := os.Symlink(filepath.Join("..", "apps", "aa", "viewdefs", "T.DEFAULT.html"), filepath.Join(siteVD, "T.DEFAULT.html")); err != nil {
		t.Skipf("cannot create symlinks: %v", err)
	}
	_, logs := loadSite(siteDir, nil)
	for _, line := range *logs {
		if strings.Contains(line, string(site.SiteSource)) {
			t.Errorf("logged a collision with a link to the winner's own file: %q", line)
		}
	}
	if len(*logs) != 1 {
		t.Errorf("logs = %q, want only the real aa/zz collision", *logs)
	}
}

// appFile is the path of an app's viewdef as the manager registers it.
func appFile(siteDir, app, key string) string {
	return filepath.Join(siteDir, "lua", app, "viewdefs", key+".html")
}

// Test: collisions are listed with both sources and files (R141, R137)
func TestCollisionsAreListed(t *testing.T) {
	siteDir := twoAppSite(t)
	want := Collision{
		Key: "T.DEFAULT", Winner: "aa", WinnerPath: appFile(siteDir, "aa", "T.DEFAULT"),
		Loser: "zz", LoserPath: appFile(siteDir, "zz", "T.DEFAULT"),
	}
	check := func(label string, m *ViewdefManager) {
		t.Helper()
		got := m.Collisions()
		if len(got) != 1 {
			t.Fatalf("%s: Collisions() = %+v, want one entry", label, got)
		}
		got[0].loserRank = 0
		if got[0] != want {
			t.Errorf("%s: collision = %+v, want %+v", label, got[0], want)
		}
	}
	m, _ := loadSite(siteDir, nil)
	check("two apps", m)

	siteVD := filepath.Join(siteDir, "viewdefs")
	os.Mkdir(siteVD, 0o755)
	if err := os.Symlink(filepath.Join("..", "apps", "aa", "viewdefs", "T.DEFAULT.html"), filepath.Join(siteVD, "T.DEFAULT.html")); err != nil {
		t.Skipf("cannot create symlinks: %v", err)
	}
	m, _ = loadSite(siteDir, nil)
	check("plus a site link to the winner", m)
}

// Test: a fixed collision drops out on re-discovery, and the log does not repeat (R142)
func TestFixedCollisionDropsOut(t *testing.T) {
	siteDir := twoAppSite(t)
	m, logs := loadSite(siteDir, nil)
	m.SetSources(m.Sources())
	if n := len(m.Collisions()); n != 1 {
		t.Errorf("after re-discovery, %d collisions listed, want 1", n)
	}
	if len(*logs) != 1 {
		t.Errorf("logs = %q, want the collision logged once", *logs)
	}
	os.Remove(filepath.Join(siteDir, "apps", "zz", "viewdefs", "T.DEFAULT.html"))
	m.SetSources(m.Sources())
	if got := m.Collisions(); len(got) != 0 {
		t.Errorf("after fixing, Collisions() = %+v, want none", got)
	}
}

// Test: a collision first met at run time joins the list (R142)
func TestRuntimeCollisionJoinsList(t *testing.T) {
	siteDir := twoAppSite(t)
	m, _ := loadSite(siteDir, nil)
	for _, app := range []string{"aa", "zz"} {
		path := appFile(siteDir, app, "V.DEFAULT")
		os.WriteFile(path, []byte("<template>"+app+"</template>"), 0o644)
		m.ApplyFileChange(path)
	}
	got := m.Collisions()
	if len(got) != 2 || got[1].Key != "V.DEFAULT" || got[1].Winner != "aa" || got[1].Loser != "zz" {
		t.Errorf("Collisions() = %+v, want T.DEFAULT then V.DEFAULT (aa over zz)", got)
	}
}

// Test: the offline resolver agrees with a loaded manager (R144)
func TestResolveCollisionsAgrees(t *testing.T) {
	siteDir := twoAppSite(t)
	m, _ := loadSite(siteDir, []string{"zz"})
	got, want := ResolveCollisions(siteDir, []string{"zz"}), m.Collisions()
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("ResolveCollisions = %+v, loaded manager = %+v", got, want)
	}
	if got[0].Winner != "zz" || got[0].Loser != "aa" {
		t.Errorf("winner %s over %s, want zz (built-in) over aa", got[0].Winner, got[0].Loser)
	}
}

// Test: losers of one key are listed in rank order (R141)
func TestCollisionLosersInRankOrder(t *testing.T) {
	siteDir := twoAppSite(t)
	vd := filepath.Join(siteDir, "apps", "mm", "viewdefs")
	os.MkdirAll(vd, 0o755)
	os.WriteFile(filepath.Join(vd, "T.DEFAULT.html"), []byte("<template>mm</template>"), 0o644)
	sitetest.LinkDir(t, filepath.Join("..", "apps", "mm"), filepath.Join(siteDir, "lua", "mm"))
	for i := 0; i < 20; i++ { // map order varies; a wrong comparator shows within a few runs
		m, _ := loadSite(siteDir, nil)
		got := m.Collisions()
		if len(got) != 2 || got[0].Loser != "mm" || got[1].Loser != "zz" {
			t.Fatalf("Collisions() losers = %+v, want mm then zz", got)
		}
	}
}
