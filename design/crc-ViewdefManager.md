# ViewdefManager
**Requirements:** R120, R125-128, R137, R141, R142, R144

Backend registry of viewdef files: which file supplies each `TYPE.NAMESPACE`
key, and which sessions have received which keys.

## Knows
- viewdefs: key → entry (content, file path, modTime, source label)
- sources: ranked viewdef sources from AppDiscovery (replaces the single
  viewdefDir)
- sessionViewdefs: session ID → keys sent
- live: the current collisions (key|loserDir → Collision), cleared on every
  LoadFromSources so it always matches the current ranking
- logged: key|loserDir pairs already logged; never cleared, so the log
  reports each collision once per server run

## Does
- LoadFromSources(sources): walk each source in rank order; register a key the
  first time it is seen. A later source offering an already-registered key is
  a collision: it is not registered, it is recorded in `live`, and it is
  logged once per run, naming the key, the winning source and the losing
  source. A later path resolving to
  the same file as the winner (os.SameFile) is skipped silently (R137).
- SetSources(sources): replace the source list (on re-discovery) and re-run
  LoadFromSources from scratch, so the winner set always matches the current
  ranking; keys whose content changed are marked for re-push.
- Collisions(): the `live` set as a list of Collision (key, winner and
  loser source names and file paths), ordered by key then loser rank (R141).
  A runtime offer that loses (ApplyFileChange, on-demand load) joins it (R142).
- ResolveCollisions(siteDir, builtins) (package function): discover the
  site's apps, build the sources, LoadFromSources into a fresh manager with a
  silent logger, and return its Collisions(): the same code a starting server
  runs, so it agrees with a live server on the same files (R144).
- tryLoadFromFilesystem(type): for a type first seen after startup, glob
  `TYPE.*.html` in each source in rank order under the same first-wins rule.
- reloadFile(path): reload one changed file; ignored if the file's key is
  owned by a different (higher-ranked) source.
- LoadFromBundle / LoadFromFS: unchanged, used when no site directory is
  configured.

## Collaborators
- AppDiscovery: provides ranked sources
- ViewdefStore (hot loader): calls reloadFile and SetSources
- Server: calls LoadFromSources during setup
- Config: Log for collision messages
- cli exports: Collision as `cli.ViewdefCollision`, ResolveCollisions as
  `cli.ResolveViewdefCollisions`

## Sequences
- seq-app-discovery.md
- seq-viewdef-hotload.md
