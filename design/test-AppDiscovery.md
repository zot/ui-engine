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

## Test: one file reached through two paths is not a collision
**Purpose:** an old per-viewdef link pointing into an app is the same viewdef, not a conflict; nothing is logged (R137)
**Input:** two-app collision site plus a site `viewdefs/T.DEFAULT.html` symlink into `aa`'s viewdef
**Expected:** only the real aa/zz collision is logged
**Refs:** crc-ViewdefManager.md
**Code:** internal/viewdef/viewdefs_test.go
**Alarm:** 10
**Fire alarm:** compare paths instead of files when deciding to log (the first version). Found live 2026-10-07: ark's restart logged 53 such "collisions", every one an app against its own old link in `~/.ark/viewdefs`. Red: a `wins over (site)` line.
**Inject:** internal/viewdef/viewdefs.go:offerLocked
**Pulled:** 2026-10-07 — rang: `logged a collision with a link to the winner's own file`

## Test: collisions are listed with both sources and files
**Purpose:** the live list names key, winner and loser sources and their files; a same-file pair is not listed (R141, R137)
**Input:** `twoAppSite`; then the same site plus a site `viewdefs/T.DEFAULT.html` linking to `aa`'s file
**Expected:** one entry `{T.DEFAULT, aa, <aa file>, zz, <zz file>}` both times
**Refs:** crc-ViewdefManager.md, seq-app-discovery.md
**Code:** internal/viewdef/viewdefs_test.go
**Alarm:** 13
**Fire alarm:** stop recording the loser into the live set in offerLocked (keep the log). Red: `Collisions()` is empty while the log still names the collision.
**Inject:** internal/viewdef/viewdefs.go:offerLocked
**Pulled:** 2026-10-13 — rang: `two apps: Collisions() = [], want one entry`

## Test: losers of one key are listed in rank order
**Purpose:** the secondary order, by the losing source's rank (R141)
**Input:** `twoAppSite` plus app `mm` shipping `T.DEFAULT.html`; load twenty times
**Expected:** losers `mm`, then `zz`, every time
**Refs:** crc-ViewdefManager.md
**Code:** internal/viewdef/viewdefs_test.go
**Alarm:** 18
**Fire alarm:** drop the loserRank tie-break in Collisions. Red: losers `zz` then `mm` on some load.
**Inject:** internal/viewdef/viewdefs.go:Collisions
**Pulled:** 2026-10-13 — rang: `losers = [… Loser:zz … Loser:mm …], want mm then zz`

## Test: a fixed collision drops out on re-discovery, and the log does not repeat
**Purpose:** the list is the current set; the log is once per run (R142)
**Input:** `twoAppSite`; SetSources with the same sources; then remove `zz`'s file and SetSources again
**Expected:** after the first SetSources, still one entry and still one log line; after the removal, no entries
**Refs:** crc-ViewdefManager.md, seq-app-discovery.md
**Code:** internal/viewdef/viewdefs_test.go
**Alarm:** 14
**Fire alarm:** do not clear the live set in loadFromSourcesLocked. Red: the removed collision is still listed.
**Inject:** internal/viewdef/viewdefs.go:loadFromSourcesLocked
**Pulled:** 2026-10-13 — rang: `after fixing, Collisions() = [{Key:T.DEFAULT …}], want none`

## Test: a collision first met at run time joins the list
**Purpose:** a hot-reloaded losing file is listed without a re-discovery (R142)
**Input:** `twoAppSite` loaded; then write `V.DEFAULT.html` into `aa` and `zz` and ApplyFileChange both, `aa` first
**Expected:** `Collisions()` lists `V.DEFAULT` with winner `aa`, loser `zz`
**Refs:** crc-ViewdefManager.md
**Code:** internal/viewdef/viewdefs_test.go
**Alarm:** 15
**Fire alarm:** record into the live set only during loadFromSourcesLocked, not in ApplyFileChange's offer. Red: `V.DEFAULT` missing from the list.
**Inject:** internal/viewdef/viewdefs.go:ApplyFileChange
**Pulled:** 2026-10-13 — rang: `Collisions() = [{Key:T.DEFAULT …}], want T.DEFAULT then V.DEFAULT`

## Test: the offline resolver agrees with a loaded manager
**Purpose:** ResolveCollisions builds the same sources with the given built-ins and returns what a loaded manager lists (R144, R143)
**Input:** `twoAppSite` with builtins `{zz}`
**Expected:** `ResolveCollisions(site, {zz})` equals `loadSite(site, {zz}).Collisions()`, winner `zz`, loser `aa`; and a running server's `ViewdefCollisions()` on the same site agrees
**Refs:** crc-ViewdefManager.md, crc-Server.md, seq-app-discovery.md
**Code:** internal/viewdef/viewdefs_test.go, internal/server/site_test.go
**Alarm:** 16
**Fire alarm:** ResolveCollisions discovers with no built-ins. Red: winner `aa`, not `zz`.
**Inject:** internal/viewdef/viewdefs.go:ResolveCollisions
**Pulled:** 2026-10-13 — rang: `ResolveCollisions = [{… Winner:aa …}], loaded manager = [{… Winner:zz …}]`

## Test: ui.viewdefCollisions returns the live list to Lua
**Purpose:** Lua sees the same entries the manager lists, with the spec's field names (R145)
**Input:** a manager loaded from `aa`/`zz` colliding on `T.DEFAULT`; a runtime built on it
**Expected:** a one-entry array whose `key`, `winner`, `winnerPath`, `loser`, `loserPath` equal the manager's entry
**Refs:** crc-LuaSession.md
**Code:** internal/lua/collisions_test.go
**Alarm:** 17
**Fire alarm:** swap `winner` and `loser` when building the row. Red: `winner = "zz", want "aa"`.
**Inject:** internal/lua/runtime.go:registerUIModule
**Pulled:** 2026-10-13 — rang: `winner = "zz", want "aa"`

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
**Fire alarm:** disable the file-identity merge in queueReload. On Linux this does not ring (EvalSymlinks resolves the symlink, so both paths already share a tracking key); it rings on Windows, where EvalSymlinks leaves junctions unresolved and the two paths carry two keys, which is the case the merge exists for. Pull it on the Windows VM.
**Inject:** internal/lua/hotloader.go:queueReload
**Pulled:** 2026-10-07 — rang on Windows (windows-gpu, arktest, scheduled task): `pending reloads = 2, want 1`
**History:** the first pull (key by changed path, Linux) rang after the watcher version was rewritten as a direct queue test; the Windows run then showed key-only dedupe fails there, and the merge by file identity replaced it.

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

## Test: an active app's whole directory is served at /<app>/
**Purpose:** `/<app>/<path>` serves `<path>` from the app's directory through its entry, including files like `app.lua`; an unknown app is not found (R138)
**Input:** app `a` with `css/x.css` and `app.lua`, linked by `lua/a`; GET `/a/css/x.css`, `/a/app.lua`, `/nosuch/x.css`
**Expected:** 200 with each file's content; 404 for `nosuch`
**Refs:** crc-HTTPEndpoint.md
**Code:** internal/server/appstatic_test.go
**Alarm:** 11
**Fire alarm:** skip the app branch in resolveStatic. Red: `/a/app.lua` is 404.
**Inject:** internal/server/http.go:resolveStatic
**Pulled:** 2026-10-07 — rang: `GET /a/app.lua = 404`

## Test: a site html/ file overrides the app's file
**Purpose:** html/ is consulted before the app, a limited user override (R139)
**Input:** the same site plus `html/a/css/x.css` with different content
**Expected:** GET `/a/css/x.css` serves the html/ copy
**Refs:** crc-HTTPEndpoint.md
**Code:** internal/server/appstatic_test.go
**Alarm:** 12
**Fire alarm:** consult the app before html/. Red: the app's copy is served.
**Inject:** internal/server/http.go:resolveStatic
**Pulled:** 2026-10-07 — rang: `GET /a/css/x.css = 200 "app", want 200 "site"`

## Test: no app serving without a site directory
**Purpose:** with no apps dir (no site directory) app paths are not found (R140)
**Input:** the same site with the endpoint's appsDir cleared
**Expected:** GET `/a/css/x.css` is 404
**Refs:** crc-HTTPEndpoint.md
**Code:** internal/server/appstatic_test.go
**Note:** no fire alarm: removing the `appsDir == ""` guard still yields 404, because the app path then resolves relative to the working directory, where no `a/` exists. The guard makes the property explicit rather than accidental; the test pins the outcome.

## Test: watching through a Windows junction
**Purpose:** confirm discovery, viewdef loading and hot reload work when `lua/<app>` is a junction made by a standard user
**Input:** every test above, with app entries made by sitetest.LinkDir (a junction on Windows); test binaries cross-compiled and run on the Windows test VM as the standard user `arktest` from a scheduled task (not ssh: RedirectionTrust refuses non-admin junctions)
**Expected:** same results as on Linux
**Ran:** 2026-10-07, windows-gpu (build 26200), arktest via `.scratch/winrun.sh -u arktest`: all pass (static-serving tests included, same day) after the queueReload identity merge; the two tests needing a *file* symlink skip (no symlink privilege). The first run caught the junction tracking-key split above.
**Refs:** crc-AppDiscovery.md
