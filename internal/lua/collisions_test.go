// CRC: crc-LuaSession.md | Test: test-AppDiscovery.md
package lua

import (
	"os"
	"path/filepath"
	"testing"

	lua "github.com/yuin/gopher-lua"
	"github.com/zot/ui-engine/internal/config"
	"github.com/zot/ui-engine/internal/site"
	"github.com/zot/ui-engine/internal/site/sitetest"
	"github.com/zot/ui-engine/internal/viewdef"
)

// Test: ui.viewdefCollisions returns the live list to Lua (R145)
func TestUIViewdefCollisions(t *testing.T) {
	siteDir := t.TempDir()
	os.MkdirAll(filepath.Join(siteDir, "lua"), 0o755)
	for _, name := range []string{"aa", "zz"} {
		vd := filepath.Join(siteDir, "apps", name, "viewdefs")
		os.MkdirAll(vd, 0o755)
		os.WriteFile(filepath.Join(vd, "T.DEFAULT.html"), []byte("<template>"+name+"</template>"), 0o644)
		sitetest.LinkDir(t, filepath.Join("..", "apps", name), filepath.Join(siteDir, "lua", name))
	}
	vdm := viewdef.NewViewdefManager()
	vdm.SetLogger(func(int, string, ...interface{}) {})
	vdm.LoadFromSources(site.ViewdefSources(siteDir, site.Discover(siteDir, nil)))

	rt, err := NewRuntime(config.DefaultConfig(), "/tmp", vdm)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	defer rt.Shutdown()

	want := vdm.Collisions()[0]
	rt.execute(func() (interface{}, error) {
		L := rt.State
		if err := L.DoString(`c = ui.viewdefCollisions()`); err != nil {
			t.Fatalf("ui.viewdefCollisions(): %v", err)
		}
		list, ok := L.GetGlobal("c").(*lua.LTable)
		if !ok || list.Len() != 1 {
			t.Fatalf("ui.viewdefCollisions() = %v, want a one-entry array", L.GetGlobal("c"))
		}
		row := list.RawGetInt(1).(*lua.LTable)
		for field, v := range map[string]string{
			"key": want.Key, "winner": want.Winner, "winnerPath": want.WinnerPath,
			"loser": want.Loser, "loserPath": want.LoserPath,
		} {
			if got := row.RawGetString(field).String(); got != v {
				t.Errorf("%s = %q, want %q", field, got, v)
			}
		}
		return nil, nil
	})
}
