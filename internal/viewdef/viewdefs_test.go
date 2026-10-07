// CRC: crc-ViewdefManager.md | Test: test-AppDiscovery.md
package viewdef

import (
	"fmt"
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
		if err := os.Symlink(filepath.Join("..", "apps", name), filepath.Join(siteDir, "lua", name)); err != nil {
			t.Skipf("cannot create symlinks: %v", err)
		}
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
