# AppDiscovery
**Requirements:** R121, R122, R123, R124, R125

Finds the active apps in a site directory and ranks them. Pure file-system
reads, no watching and no state: every caller (viewdef loading, both hot
loaders) asks again when it needs a fresh answer.

## Knows
- siteDir: the site directory (`config.Server.Dir`)
- builtins: set of built-in app names (`config.Server.BuiltinApps`)

## Does
- Discover(siteDir, builtins) []App: list `lua/*`, keep entries that resolve
  to a directory (os.Stat, which reads through a symlink or junction; never
  Lstat or a link test), rank them built-in tier first, then user tier,
  alphabetical by name within each. A builtin name with no active entry is
  skipped.
- App.Dir: the app's directory as seen *through* its entry
  (`<siteDir>/lua/<app>`), never the resolved target, so every path built from
  it (entry file, modules, viewdefs, watches) stays under the entry.
- ViewdefSources(siteDir, apps) []Source: each app's `<Dir>/viewdefs` in rank
  order, then the site's `<siteDir>/viewdefs` last. A source whose directory
  does not exist is omitted silently.

## Collaborators
- ViewdefManager: loads viewdefs from ViewdefSources
- ViewdefStore (hot loader): re-discovers when `lua/` changes
- LuaHotLoader: watches each App.Dir
- Config: siteDir and builtins

## Sequences
- seq-app-discovery.md
