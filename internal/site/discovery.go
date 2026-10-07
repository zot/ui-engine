// Package site finds the active apps in a site directory and ranks them.
// CRC: crc-AppDiscovery.md | Seq: seq-app-discovery.md
// Spec: app-discovery.md
package site

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SiteSource labels the site's own viewdefs directory, ranked below every app.
const SiteSource = "(site)"

// App is an active app. Dir is the app's directory seen through its
// lua/<app> entry, never the resolved target, so every path built from it
// stays under the entry.
type App struct {
	Name    string
	Dir     string
	Builtin bool
}

// Source is one viewdef directory, labeled by the app that owns it.
type Source struct {
	Label string
	Dir   string
}

// Discover lists the active apps under siteDir/lua, built-in tier first,
// then user apps, alphabetical within each tier.
// CRC: crc-AppDiscovery.md | Seq: seq-app-discovery.md#1.2 | R121, R123, R124
func Discover(siteDir string, builtins []string) []App {
	luaDir := filepath.Join(siteDir, "lua")
	entries, err := os.ReadDir(luaDir)
	if err != nil {
		return nil
	}
	isBuiltin := make(map[string]bool, len(builtins))
	for _, name := range builtins {
		isBuiltin[name] = true
	}
	var apps []App
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		dir := filepath.Join(luaDir, name)
		// R122: read through the entry; a junction reports as a directory,
		// a symlink only through Stat, so never test for a link.
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			continue
		}
		apps = append(apps, App{Name: name, Dir: dir, Builtin: isBuiltin[name]})
	}
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Builtin != apps[j].Builtin {
			return apps[i].Builtin
		}
		return apps[i].Name < apps[j].Name
	})
	return apps
}

// ViewdefSources returns each app's viewdefs directory in rank order, then
// the site's own. Directories that do not exist are omitted silently.
// CRC: crc-AppDiscovery.md | Seq: seq-app-discovery.md#1.3 | R120, R125
func ViewdefSources(siteDir string, apps []App) []Source {
	var sources []Source
	add := func(label, dir string) {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			sources = append(sources, Source{Label: label, Dir: dir})
		}
	}
	for _, app := range apps {
		add(app.Name, filepath.Join(app.Dir, "viewdefs"))
	}
	add(SiteSource, filepath.Join(siteDir, "viewdefs"))
	return sources
}
