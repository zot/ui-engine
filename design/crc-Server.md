# Server
**Requirements:** R119, R120, R138, R143, R153, R168-171, R193

Top-level server. This card covers its site-directory responsibilities and its
session entry points (references and dangerous execution); protocol and HTTP behavior
are on their own cards.

## Knows
- config: Config (site directory, built-in app names, hotload flag)
- viewdefManager: ViewdefManager
- luaHotLoader / viewdefHotLoader: optional, when hot-loading
- luaSessions: vended ID to LuaSession; SessionRef resolves through it (R153, R173)
- dangerousSites: set of call sites already warned about, mutex-guarded (R170)

## Does
- Start: when a site directory is configured and does not exist, return an
  error naming the path before listening; never panic.
- setupViewdefs: discover apps (AppDiscovery), load viewdefs from the ranked
  sources (ViewdefManager.LoadFromSources); a missing site `viewdefs/` is not
  logged. Starts the viewdef hot loader when hot-loading.
- setupSite: sets the HTTP endpoint's static dir (`<site>/html`) and apps dir (`<site>/lua`) (R138, R140)
- ViewdefCollisions(): the running server's viewdef collisions, read from
  ViewdefManager.Collisions() (R143).
- setupLua: Lua directory is `<site>/lua`; starts the Lua hot loader when
  hot-loading.
- SessionRef(vendedID): a reference holding a weak pointer to the registered
  LuaSession, or false when none is registered (R153).
- ExecuteInSession: delegates to PrimitiveExecuteInSession with dangerous true (R168).
- DestroyLuaBackendForSession: first closes every open Tailor on the session as
  failed (R193).
- PrimitiveExecuteInSession(vendedID, fn, dangerous): when dangerous, find the site that
  called ExecuteInSession; with AllowDangerousExecution set, warn once per site and
  continue (R170); unset, panic naming the site before queueing (R171). Then the
  behavior ExecuteInSession had before: resolve IDs, queue through the endpoint and the
  LuaSession, wait (R169).

## Collaborators
- AppDiscovery, ViewdefManager, ViewdefStore (hot loader), LuaHotLoader, Config, SessionRef, WebSocketEndpoint, LuaSession

## Sequences
- seq-app-discovery.md
- seq-server-startup.md
- seq-session-ref.md
- seq-tailor.md
