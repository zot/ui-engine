# CRC: Config

**Spec:** deployment.md

## Responsibilities

| Does                              | Knows                         |
|-----------------------------------|-------------------------------|
| Load config from TOML file        | Config file path              |
| Merge with CLI flags and env vars | Default values                |
| Provide typed access to settings  | Current configuration state   |
| Validate configuration values     | Valid option ranges           |
| Get platform-specific defaults    | Platform type (POSIX/Windows) |
| Provide verbosity level           | Verbosity level (0-3)         |

## Collaborators

| Collaborator      | Interaction                                   |
|-------------------|-----------------------------------------------|
| EmbeddedSite      | Reads config.toml from embedded archive       |
| StorageBackend    | Provides storage type and connection settings |
| LuaRuntime        | Provides Lua enabled flag and path            |
| SessionManager    | Provides session timeout setting              |
| WebSocketEndpoint | Provides host, port, and verbosity (level 1)  |
| HTTPEndpoint      | Provides host and port settings               |
| BackendSocket     | Provides socket path and verbosity (level 1)  |
| ProtocolHandler   | Provides verbosity for message logging (level 2) |
| VariableStore     | Provides verbosity for operation logging (level 3) |

## Configuration Options

| Option          | CLI                 | Env                  | TOML                | Default |
|-----------------|---------------------|----------------------|---------------------|---------|
| Lua enabled     | `--lua` / `--no-lua`| `UI_LUA`             | `lua.enabled`       | `true`  |
| Session timeout | `--session-timeout` | `UI_SESSION_TIMEOUT` | `session.timeout`   | `"24h"` |
| Socket path     | `--socket`          | `UI_SOCKET`          | `backend.socket`    | platform-specific |
| Verbosity       | `-v`, `-vv`, `-vvv` | `UI_VERBOSITY`       | `logging.verbosity` | `0`     |

## Sequences

- seq-server-startup.md (load configuration)

## Notes

- Configuration priority: CLI flags > env vars > config.toml > defaults
- Missing config.toml is not an error (uses defaults)
- TOML parsing uses standard Go TOML library
- Socket default: POSIX `/tmp/ui.sock`, Windows `\\.\pipe\ui`
- Session timeout: How long session persists without activity (default 24h, 0=never)
- Frontend can reconnect to any session that hasn't timed out
- Verbosity levels control debug output (0=none, 1=connections, 2=protocol, 3=variables)
- Components query Config.Verbosity() to decide what to log

**Backend Modes (see interfaces.md):**
- **Embedded Lua only** (`--lua`, no backend connection): Lua creates variable 1
- **Connected backend only** (`--no-lua`): Backend creates variable 1
- **Hybrid** (`--lua` + backend connects): Developer decides where variable 1 is created
