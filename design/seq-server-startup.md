# Sequence: Server Startup

**Source Spec:** deployment.md
**Use Case:** UI server process initialization and configuration loading

## Participants

- Main: Server entry point
- Config: Configuration loader
- EmbeddedSite: Bundled site archive
- StorageBackend: Storage layer
- LuaRuntime: Lua VM
- BackendSocket: Backend API socket
- WebSocketEndpoint: WebSocket handler
- HTTPEndpoint: HTTP handler

## Sequence

```
     Main             Config         EmbeddedSite      StorageBackend       LuaRuntime        BackendSocket     WebSocketEndpoint    HTTPEndpoint
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
        |                 |                 |                 |<--loadScripts()--|                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |<--luaRuntime----------------------------------------------|                 |                 |                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |--listen(socketPath)------------------------------------------------------->|                 |                 |
        |                 |                 |                 |                 |                 |                 |                 |
        |<--backendSocket-----------------------------------------------------------|                 |                 |
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
- Lua runtime only initialized if enabled in config
- BackendSocket listens on platform-specific path (POSIX: /tmp/ui.sock, Windows: \\.\pipe\ui)
- Socket path configurable via --socket, UI_SOCKET, or server.socket
- Server starts listening after all subsystems ready
