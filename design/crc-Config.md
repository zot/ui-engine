# CRC: Config

**Spec:** deployment.md, config.md

## Responsibilities

| Does                              | Knows                         |
|-----------------------------------|-------------------------------|
| Load config from TOML file        | Config file path              |
| Merge with CLI flags and env vars | Default values                |
| Provide typed access to settings  | Current configuration state   |
| Validate configuration values     | Valid option ranges           |
| Get platform-specific defaults    | Platform type (POSIX/Windows) |
| Provide centralized logging       | Verbosity level (0-4)         |
| Log: Log message with level check | Logging configuration         |

## Collaborators

| Collaborator      | Interaction                                   |
|-------------------|-----------------------------------------------|
| EmbeddedSite      | Reads config.toml from embedded archive       |
| LuaRuntime        | Provides Lua enabled flag and path            |
| SessionManager    | Provides session timeout setting              |
| WebSocketEndpoint | Logging delegate (connection events)          |
| HTTPEndpoint      | Provides host and port settings               |
| BackendSocket     | Logging delegate (socket events)              |
| ProtocolHandler   | Logging delegate (protocol messages)          |
| VariableStore     | Logging delegate (variable operations)        |

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
- **Centralized Logging**: All components must use `Config.Log()` for output.
- Verbosity levels:
  - 0: Errors only
  - 1: Connections
  - 2: Protocol messages
  - 3: Variable operations
  - 4: Variable values