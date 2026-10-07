# ViewdefManager
**Requirements:** R120, R125-128, R137

Backend registry of viewdef files: which file supplies each `TYPE.NAMESPACE`
key, and which sessions have received which keys.

## Knows
- viewdefs: key → entry (content, file path, modTime, source label)
- sources: ranked viewdef sources from AppDiscovery (replaces the single
  viewdefDir)
- sessionViewdefs: session ID → keys sent

## Does
- LoadFromSources(sources): walk each source in rank order; register a key the
  first time it is seen. A later source offering an already-registered key is
  a collision: it is not registered, and the collision is logged once, naming
  the key, the winning source and the losing source. A later path resolving to
  the same file as the winner (os.SameFile) is skipped silently (R137).
- SetSources(sources): replace the source list (on re-discovery) and re-run
  LoadFromSources from scratch, so the winner set always matches the current
  ranking; keys whose content changed are marked for re-push.
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

## Sequences
- seq-app-discovery.md
- seq-viewdef-hotload.md
