# Sequence: Server Startup

**Source Spec:** deployment.md
**Use Case:** UI server process initialization and configuration loading

## Participants

- Main: Server entry point
- Config: Configuration loader
- EmbeddedSite: Bundled site archive
- LuaSession: Per-session Lua environment (created per frontend session)
- SessionManager: Session management
- WebSocketEndpoint: WebSocket handler
- HTTPEndpoint: HTTP handler

## Sequence

```
     Main             Config         EmbeddedSite      LuaSession       SessionManager   WebSocketEndpoint    HTTPEndpoint
        |                 |                 |                 |                 |                 |                 |
        |--parseCLI()---->|                 |                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |--loadEnv()----->|                 |                 |                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |                 |--readConfig()-->|                 |                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |                 |<-config.toml----|                 |                 |                 |                 |                 |
        |                 |  (or nil)       |                 |                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |                 |--parseToml()--->|                 |                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |                 |--mergeConfig()->|                 |                 |                 |                 |                 |
        |                 |  (cli>env>file) |                 |                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |<--config--------|                 |                 |                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |--[if lua.enabled]                 |                 |                 |                 |                 |                 |
        |  initLuaFactory(path)---------------------------------->|                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |                 |                 |                 |  [factory ready] |                 |                 |                 |
        |                 |                 |                 |  [NO sessions yet]                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |<--luaFactory-----------------------------------------------|                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |--initSessionManager(luaFactory)---------------------------------------->|                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |<--sessionManager------------------------------------------------------|                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |--listen(host,port)-------------------------------------------------------------------------->|                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |--listen(host,port)------------------------------------------------------------------------------------------>|
        |                 |                 |                 |                 |                 |                 |                 |
        |--[ready to accept connections]---                                                                            |
        |                 |                 |                 |                 |                 |                 |                 |
```

## Notes

- Config priority: CLI flags > environment variables > config.toml > defaults
- Missing config.toml uses defaults (not an error)
- Lua runtime only initialized if enabled in config (--lua, default: true)
- **Embedded Lua only**: External backend sockets removed; all backend logic runs in embedded Lua
- **Session-based Lua**: Lua factory is ready at startup but does NOT create any LuaSessions yet
- **Variable 1 per session**: Each LuaSession creates variable 1 when main.lua runs (see seq-lua-session-init.md)
- **Executor channel**: Each LuaSession has its own executor for single-threaded Lua access
- SessionManager is linked to Lua factory for creating LuaSessions when frontend sessions are created
- Server starts listening after all subsystems ready
- Verbosity level (0-4) loaded from -v flags, UI_VERBOSITY, or logging.verbosity
- Components receive Config for centralized logging (delegating log calls to Config)
