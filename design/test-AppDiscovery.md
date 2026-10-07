# Test Design: App Discovery and Per-App Hot-Loading
**Source:** crc-AppDiscovery.md, crc-ViewdefManager.md, crc-Server.md, crc-LuaHotLoader.md, crc-ViewdefStore.md

All cases build a site in `t.TempDir()`: `apps/<name>/` directories, `lua/<name>`
symlinks to them, optional `viewdefs/`. Discovery and collision cases need no
server and no watcher.

## Test: discovery keeps directory entries only
**Purpose:** an app is an entry that resolves to a directory; files and broken links are not apps (R121)
**Input:** `lua/main.lua` (file), `lua/a` → `apps/a`, `lua/gone` → missing target
**Expected:** Discover returns exactly `a`, with Dir `<site>/lua/a`
**Refs:** crc-AppDiscovery.md, seq-app-discovery.md
**Code:** internal/site/discovery_test.go
**Alarm:** 1
**Fire alarm:** test the entry with `Lstat` + `ModeSymlink` instead of `Stat` + `IsDir`; on Linux a symlinked app still passes, so the case must also assert that a plain directory entry `lua/plain/` counts as an app (the Windows-junction shape, which Go reports as a directory). Red: `plain` missing.
**Inject:** internal/site/discovery.go:Discover
**Pulled:** 2026-10-07 — rang: `apps = [a gone], want [a plain]`

## Test: ranking puts built-ins first, alphabetical within tiers
**Purpose:** order is built-in tier, then user tier, alphabetical inside each; inactive built-in names ignored (R123, R124)
**Input:** active `zeta`, `alpha`, `mcp`, `ark`; builtins `{mcp, ark, missing}`
**Expected:** `ark, mcp, alpha, zeta`
**Refs:** crc-AppDiscovery.md
**Code:** internal/site/discovery_test.go
**Alarm:** 2
**Fire alarm:** sort all apps alphabetically, ignoring tiers. Red: `alpha` ranks before `ark`.
**Inject:** internal/site/discovery.go:Discover
**Pulled:** 2026-10-07 — rang: `order = [alpha ark mcp zeta], want [ark mcp alpha zeta]`

## Test: viewdef sources end with the site directory, missing dirs omitted
**Purpose:** app viewdefs in rank order, then the site's; an app or site with no `viewdefs/` is skipped silently (R120, R125)
**Input:** apps `a` (has viewdefs), `b` (none); site `viewdefs/` present, then absent
**Expected:** `[lua/a/viewdefs, viewdefs]`, then `[lua/a/viewdefs]`; no log output either time
**Refs:** crc-AppDiscovery.md
**Code:** internal/site/discovery_test.go

## Test: collision goes to the higher-ranked source and is logged
**Purpose:** first-wins by rank, regardless of walk order; one log line naming key, winner, loser (R126, R127)
**Input:** user apps `aa` and `zz` both ship `T.DEFAULT.html` with different content; also builtin `zz`
**Expected:** without builtins, `aa`'s content registered, log names `T.DEFAULT`, `aa`, `zz`; with `zz` built-in, `zz` wins
**Refs:** crc-ViewdefManager.md, seq-app-discovery.md
**Code:** internal/viewdef/viewdefs_test.go
**Alarm:** 3
**Fire alarm:** let a later source overwrite an existing key (the old last-writer-wins `m.viewdefs[key] = ...`): disable the rank check in offerLocked. Red: `zz` content registered without builtins, no collision logged.
**Inject:** internal/viewdef/viewdefs.go:offerLocked
**Pulled:** 2026-10-07 — rang: `T.DEFAULT = zz`, `collision log = []`

## Test: on-demand loading honors the same rule
**Purpose:** a type first requested after startup resolves across sources in rank order (R128)
**Input:** load with sources, then add `U.DEFAULT.html` to both apps on disk and request type `U`
**Expected:** the higher-ranked app's `U.DEFAULT` is returned
**Refs:** crc-ViewdefManager.md
**Code:** internal/viewdef/viewdefs_test.go
**Alarm:** 4
**Fire alarm:** search only the last source. (Reversing the order does *not* ring: offerLocked's rank check still lets the higher-ranked file win, so the order of the search alone is not what this guards.) Red: lower-ranked content returned.
**Inject:** internal/viewdef/viewdefs.go:tryLoadFromFilesystem
**Pulled:** 2026-10-07 — rang: `on-demand U.DEFAULT = zz`

## Test: a change to the losing copy leaves the winner
**Purpose:** at hot-reload time a lower-ranked source's file cannot displace the registered winner (R126)
**Input:** two-app collision; rewrite the loser's `T.DEFAULT.html` and call ApplyFileChange on its entry path
**Expected:** ApplyFileChange reports not ok; `T.DEFAULT` keeps the winner's content
**Refs:** crc-ViewdefManager.md
**Code:** internal/viewdef/viewdefs_test.go
**Alarm:** 5
**Fire alarm:** in ApplyFileChange, register the changed file unconditionally instead of through offerLocked. Found by probing past the alarm list after simplification: with this injected, the whole suite stayed green. Red: loser accepted and its content registered.
**Inject:** internal/viewdef/viewdefs.go:ApplyFileChange
**Pulled:** 2026-10-07 — rang: `ApplyFileChange accepted the losing source's file`

## Test: missing site directory is a start error
**Purpose:** Start returns an error naming the path; no panic (R119)
**Input:** config with `Server.Dir` set to a nonexistent path
**Expected:** Start returns a non-nil error containing the path
**Refs:** crc-Server.md
**Code:** internal/server/site_test.go
**Alarm:** 6
**Fire alarm:** delete the existence check. Red: Start returns nil (or the test panics/hangs listening; the case uses port 0 and stops the server).
**Inject:** internal/server/server.go:configureHTTP
**Pulled:** 2026-10-07 — rang: `StartAsync succeeded at http://127.0.0.1:…`

## Test: app module outside the site hot-reloads
**Purpose:** a non-entry `.lua` file of an app whose directory lives outside the site reloads (R132, R134)
**Input:** app dir outside the site, `lua/a` → it; a session has loaded `a/mod.lua` via require; modify `mod.lua` in place
**Expected:** reloadInSession called for `mod.lua`'s tracking key
**Refs:** crc-LuaHotLoader.md, seq-lua-hotload.md
**Code:** internal/lua/appwatch_test.go
**Alarm:** 7
**Fire alarm:** watch only `lua/` (no per-app watches), which is today's behavior for modules. Red: no reload.
**Inject:** internal/lua/hotloader.go:Start
**Pulled:** 2026-10-07 — rang: `module of an app outside the site was not reloaded`

## Test: one file reached by two watched paths reloads once
**Purpose:** debounce is per tracking key (R134)
**Input:** `apps/a` real dir under site, `lua/a` → `apps/a`; queue a reload for `apps/a/app.lua` and for `lua/a/app.lua` directly (no watcher)
**Expected:** exactly one pending reload
**Note:** first written as a watcher test, it passed with the guard removed: on Linux inotify returns one watch for two paths naming the same directory, so only one event arrives. Windows watches each path separately and does deliver two, which is the case the guard exists for.
**Refs:** crc-LuaHotLoader.md
**Code:** internal/lua/appwatch_test.go
**Alarm:** 8
**Fire alarm:** key pendingReloads by changed path. Red: two reloads.
**Inject:** internal/lua/hotloader.go:queueReload
**Pulled:** 2026-10-07 — rang: `pending reloads = 2, want 1` (after the rewrite; the watcher version stayed green)

## Test: activating and deactivating an app re-runs discovery
**Purpose:** a new `lua/<app>` entry gets watched and its viewdefs registered; removing it drops both, but loaded Lua stays (R129, R130)
**Input:** running hot loaders; create `lua/b` → `apps/b` with `B.DEFAULT.html`; then remove it
**Expected:** after create, `B.DEFAULT` registered and a change to `apps/b/viewdefs/B.DEFAULT.html` re-pushes; after remove, `B.DEFAULT` gone, no reload for `b` files, session's loaded `b` code untouched
**Refs:** crc-ViewdefStore.md, crc-LuaHotLoader.md, seq-app-discovery.md
**Code:** internal/viewdef/hotloader_test.go, internal/lua/appwatch_test.go
**Alarm:** 9
**Fire alarm:** ignore directory create events in `lua/`. Red: `B.DEFAULT` not registered after create.
**Inject:** internal/viewdef/hotloader.go:handleEvent
**Pulled:** 2026-10-07 — rang: `B.DEFAULT not registered after activation`

## Test: site viewdefs watched only if present at startup
**Purpose:** R131/R133
**Input:** start with site `viewdefs/` present, modify a file in it; separately start without it, then create it
**Expected:** first case reloads; second case registers nothing until restart, and logs nothing
**Refs:** crc-ViewdefStore.md
**Code:** internal/viewdef/hotloader_test.go

## Test: watching through a Windows junction (manual)
**Purpose:** confirm fsnotify reports events under the junction path on Windows, as measured on Linux 2026-10-07 for symlinks
**Input:** on the Windows test VM, from a scheduled task (not ssh: RedirectionTrust refuses non-admin junctions), run the hot-loader tests with `lua/<app>` as a junction
**Expected:** same results as on Linux
**Refs:** crc-AppDiscovery.md
