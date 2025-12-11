# Sequence: Server Startup

**Source Spec:** deployment.md
**Use Case:** UI server process initialization and configuration loading

## Participants

- Main: Server entry point
- Config: Configuration loader
- EmbeddedSite: Bundled site archive
- StorageBackend: Storage layer
- LuaRuntime: Lua runtime manager
- SessionManager: Session management
- WebSocketEndpoint: WebSocket handler
- HTTPEndpoint: HTTP handler

## Sequence

```
     Main             Config         EmbeddedSite      StorageBackend       LuaRuntime       SessionManager   WebSocketEndpoint    HTTPEndpoint
        |                 |                 |                 |                 |                 |                 |                 |
        |--parseCLI()---->|                 |                 |                 |                 |                 |                 |
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
        |--initStorage(type,path)---------->|                 |                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |                 |                 |<--storage-------|                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |--[if lua.enabled]                 |                 |                 |                 |                 |                 |
        |  initLua(path)---------------------------------------->|                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |                 |                 |                 |  [starts executor]                 |                 |                 |
        |                 |                 |                 |  [NO var 1 yet]  |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |<--luaRuntime----------------------------------------------|                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |--initSessionManager(luaRuntime)---------------------------------------->|                 |                 |
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
- Storage initialized based on config type (memory/sqlite/postgresql)
- Lua runtime only initialized if enabled in config (--lua, default: true)
- **Embedded Lua only**: External backend sockets removed; all backend logic runs in embedded Lua
- **Session-based Lua**: LuaRuntime starts executor but does NOT create variable 1 at startup
- **Variable 1 per session**: Each LuaSession creates variable 1 when main.lua runs (see seq-lua-session-init.md)
- **Executor channel**: Ensures single-threaded Lua access (Lua VMs are not thread-safe)
- SessionManager is linked to LuaRuntime for creating Lua sessions
- Server starts listening after all subsystems ready
- Verbosity level (0-3) loaded from -v flags, UI_VERBOSITY, or logging.verbosity
- Components check verbosity to enable debug logging (1=connections, 2=protocol, 3=variables)
