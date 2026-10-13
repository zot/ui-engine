# App Discovery

How the server finds the apps in a site directory, the order it ranks
them in, and how that order settles viewdef-name collisions. Applies when
a site directory is configured (`--dir`, `config.Server.Dir`); a server
running from its embedded bundle is unaffected.

Language: Go. Environment: Linux, macOS, Windows.

## The site directory

The site directory must exist. If it is configured and missing, the server
refuses to start and returns an error naming the path. It does not panic:
a host embedding the server decides what a failed start means.

Inside it, `lua/` holds `main.lua` and the app entries, `viewdefs/` holds
the site's own viewdefs, and `html/` holds static files. `viewdefs/` is
optional: a site whose viewdefs all come from apps has none, and its absence
is not logged.

## Activation

An app is **active** when `lua/<app>` exists and resolves to a directory.
That directory is the app's directory, and everything the server reads for
the app is read through that one entry. Plain files directly in `lua/`
(such as `main.lua`) are not apps.

The entry may be a symlink (Unix) or a directory junction (Windows); the
server reads through it and never tests whether it is a link. On Windows,
Go reports a junction as a directory rather than as a symlink, so a link
test would find no apps there.

## Order

Active apps are ranked in **tiers**, and alphabetically by name within a
tier:

1. **Built-in apps**, named by a list the host supplies in the
   server's configuration (programmatic only, not read from the config
   file). With no list, there are no built-in apps. A listed name that is
   not active is ignored.
2. **User apps**: every other active app.
3. **The site**: the site directory's own `viewdefs/`, ranked below every
   app.

## Viewdefs

The server registers viewdefs from each active app's `viewdefs/`
directory and from the site's `viewdefs/`, in the order above. A viewdef's
name is its file name without `.html` (`TYPE.NAMESPACE`). An app with no
`viewdefs/` directory contributes none, silently.

**Collisions.** When two sources ship a viewdef of the same name, the one
ranked earlier wins and the later one is not registered. The outcome never
depends on load order, link order, or file-system walk order. Each
collision is logged, naming the viewdef, the winner and the loser.
Two paths that lead to the same file (an old per-viewdef link pointing
into an app, say) are one viewdef, not a collision: nothing is logged.

On-demand loading (a viewdef for a new `type` requested after startup)
searches the same sources in the same order and applies the same rule.

**Listing collisions.** A host can ask for the collisions as a list, so it
can show them to a user and not only log them. Each entry names the
viewdef, the winning source and its file, and the losing source and its
file. Entries are ordered by viewdef name, then by the losing source's rank.

The list is the *current* set. Re-discovery recomputes it, so a collision
the user has fixed drops out, and a collision that first appears at run time
(a hot-reloaded file whose name a higher-ranked source already owns) joins
it. The log is different: it reports each collision once per server run, and
does not repeat itself when re-discovery finds the same collision again.

There are two ways to get the list, and both run the same registration code,
so they agree for the same files:

- `(*cli.Server).ViewdefCollisions() []cli.ViewdefCollision`: the
  collisions of a running server. Lua code reads the same list with
  `ui.viewdefCollisions()`.
- `cli.ResolveViewdefCollisions(siteDir string, builtins []string)
  []cli.ViewdefCollision`: computed from disk without a server, by
  discovering the site's apps with the given built-in list and registering
  their viewdefs exactly as a starting server would. A host uses this when
  no server is running.

`cli.ViewdefCollision` has the fields `Key`, `Winner`, `WinnerPath`,
`Loser`, and `LoserPath`. A source's name is its app name, or `(site)` for
the site's own `viewdefs/`.

## Static files

Each active app's whole directory is served over HTTP at `/<app>/…`:
`/<app>/css/panel.css` is the file `css/panel.css` in that app's directory,
read through its `lua/<app>` entry. Everything in the directory is
reachable; an app marks what is internal by convention (a leading `_`), not
by hiding it. This replaces linking an app into `html/` to publish its files.

A request is resolved in this order:

1. A session path, as before.
2. The site's `html/`: if `html/<path>` exists, it is served. This lets a
   user override any app file by placing one in `html/`.
3. An active app named by the first path segment: the rest of the path is
   served from that app's directory.

Anything else is not found. There is no app serving without a site
directory.

## Re-discovery

With hot-loading enabled, the server watches `lua/`. When an app is
activated or deactivated there, it re-runs discovery: it updates the
per-app watches, re-registers viewdefs, and re-resolves collisions. Lua
code already loaded from a deactivated app stays loaded until the session
ends; deactivation stops new loads and hot reloads, it does not unload.

A site `viewdefs/` directory created or removed while the server runs is
not noticed until the next start.

See [Hot-Loading System](main.md#hot-loading-system) for how changes inside
an app are reloaded.
