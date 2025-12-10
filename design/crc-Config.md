# CRC: Config

**Spec:** deployment.md

## Responsibilities

| Does                              | Knows                       |
|-----------------------------------|-----------------------------|
| Load config from TOML file        | Config file path            |
| Merge with CLI flags and env vars | Default values              |
| Provide typed access to settings  | Current configuration state |
| Validate configuration values     | Valid option ranges         |
| Get platform-specific defaults    | Platform type (POSIX/Windows) |

## Collaborators

| Collaborator      | Interaction                                   |
|-------------------|-----------------------------------------------|
| EmbeddedSite      | Reads config.toml from embedded archive       |
| StorageBackend    | Provides storage type and connection settings |
| LuaRuntime        | Provides Lua enabled flag and path            |
| SessionManager    | Provides session timeout setting              |
| Session           | Provides connection timeout setting for grace period |
| WebSocketEndpoint | Provides host, port, and connection timeout settings |
| HTTPEndpoint      | Provides host and port settings               |
| BackendSocket     | Provides socket path (--socket, UI_SOCKET, server.socket) |

## Configuration Options

| Option             | CLI                      | Env                       | TOML                 | Default |
|--------------------|--------------------------|---------------------------|----------------------|---------|
| Session timeout    | `--session-timeout`      | `UI_SESSION_TIMEOUT`      | `session.timeout`    | `"24h"` |
| Connection timeout | `--connection-timeout`   | `UI_CONNECTION_TIMEOUT`   | `session.connection` | `"5s"`  |

## Sequences

- seq-server-startup.md (load configuration)
- seq-frontend-reconnect.md (uses connection timeout)

## Notes

- Configuration priority: CLI flags > env vars > config.toml > defaults
- Missing config.toml is not an error (uses defaults)
- TOML parsing uses standard Go TOML library
- Socket default: POSIX `/tmp/ui.sock`, Windows `\\.\pipe\ui`
- Connection timeout: Grace period for frontend reconnection after disconnect (default 5s)
- Session timeout: How long session persists without activity (default 24h, 0=never)
