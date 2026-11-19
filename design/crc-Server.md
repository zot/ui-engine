# Server
**Requirements:** R119, R120, R138, R143, R153, R168-171, R193, R118, R229, R231, R232, R233, R235, R236, R239, R254-R256, R260, R264

Top-level server. This card covers its site-directory responsibilities and its
session entry points (references and dangerous execution); protocol and HTTP behavior
are on their own cards.

## Knows
- config: Config (site directory, built-in app names, hotload flag)
- viewdefManager: ViewdefManager
- luaHotLoader / viewdefHotLoader: optional, when hot-loading
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
- SessionRef(vendedID): a reference holding a weak pointer to the session's
  LuaSession, found through luaSessionFor, or false when there is none (R153).
- luaSessionFor(vendedID): SessionManager.GetByVendedID, then the session's LuaSession;
  nil when there is no such session, no Lua session is attached yet, or it has shut down.
  The one way the server finds a Lua session; it holds none of its own (R229, R231, R232).
- GetLuaSession(vendedID): luaSessionFor, signature unchanged (R232). HandleFrontendCreate,
  HandleFrontendUpdate and the debug data provider use luaSessionFor; AfterBatch, which
  has the internal ID, asks the Session directly (R232).
- GetSessionIDs / getLuaSessions: from SessionManager.Snapshot, the sessions whose Lua
  session is attached and running (R233).
- Shutdown: SessionManager.Snapshot (the registry lock is released by then), then each
  attached Lua session's Shutdown (R118, R233). Afterwards luaSessionFor finds nothing,
  since every Lua session reports shut down.
- CreateLuaBackendForSession: attaches the new LuaSession to the Session before
  CreateLuaSession runs main.lua (R230), and installs its executor with
  `LuaSession.SetExecutor`: run posts to the session's ChanSvc and waits, without change
  detection; closeWhenDrained waits on that ChanSvc's drained signal (R254, R260); then defines `ui.execAsync` on it, so main.lua
  can use it (R250, crc-ExecAsync.md); neither the Lua session nor the backend is copied
  to the store adapter, which finds the backend through the registry (R231, R239).
- ExecuteInSession: delegates to PrimitiveExecuteInSession with dangerous true (R168).
- ExecuteInSessionAsync(vendedID, fn): SessionRef(vendedID), then Exec().PostUpdate;
  nothing when there is no such session (R235).
- PushViewdefs(vendedID, viewdefs) and triggerSessionRefresh(vendedID): post an empty function
  with the session's ExecRig.PostUpdate so change detection pushes what changed; neither
  waits, so neither is a dangerous call (R264).
- DestroyLuaBackendForSession: first closes every open Tailor on the session as
  failed (R193). Then shuts down the session's attached LuaSession, which returns at once
  and closes its VM once the session executor drains; there is no map entry to remove
  (R231, R260).
- PrimitiveExecuteInSession(vendedID, fn, dangerous): when dangerous, find the site that
  called ExecuteInSession; with AllowDangerousExecution set, warn once per site and
  continue (R170); unset, panic naming the site before queueing (R171). Then the
  behavior ExecuteInSession had before: find the session with luaSessionFor, the same way a
  reference does, queue on the endpoint (the session executor) and run fn there through
  LuaSession.RunInSession, in place, wait (R169, R236, R255).

## Collaborators
- AppDiscovery, ViewdefManager, ViewdefStore (hot loader), LuaHotLoader, Config, SessionRef, WebSocketEndpoint, LuaSession, Session, SessionManager

## Sequences
- seq-app-discovery.md
- seq-server-startup.md
- seq-session-ref.md
- seq-tailor.md
