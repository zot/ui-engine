# Server
**Requirements:** R119, R120, R138

Top-level server. This card covers only its site-directory responsibilities;
session, protocol and HTTP behavior are on their own cards.

## Knows
- config: Config (site directory, built-in app names, hotload flag)
- viewdefManager: ViewdefManager
- luaHotLoader / viewdefHotLoader: optional, when hot-loading

## Does
- Start: when a site directory is configured and does not exist, return an
  error naming the path before listening; never panic.
- setupViewdefs: discover apps (AppDiscovery), load viewdefs from the ranked
  sources (ViewdefManager.LoadFromSources); a missing site `viewdefs/` is not
  logged. Starts the viewdef hot loader when hot-loading.
- setupSite: sets the HTTP endpoint's static dir (`<site>/html`) and apps dir (`<site>/lua`) (R138, R140)
- setupLua: Lua directory is `<site>/lua`; starts the Lua hot loader when
  hot-loading.

## Collaborators
- AppDiscovery, ViewdefManager, ViewdefStore (hot loader), LuaHotLoader, Config

## Sequences
- seq-app-discovery.md
- seq-server-startup.md
