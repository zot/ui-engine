# Architecture

```
[browser] <--WebSocket--> [UI Server] <--> [backend program(s) / MCP]
                               |
                           [storage]

[browser] <--HTTP--> [web server] <--FastCGI--> [UI Instance] <--> [UI Server]
                                                                        |
                                                                    [storage]
```

## Deployment Modes

- **Standalone server**: WebSocket + HTTP, handles browsers and backends directly
- **FastCGI**: Ephemeral UI instances (behind nginx, Apache, etc.) communicate with a persistent UI Server
- **Command-line program**: For scripting and testing, can act as a client to a UI Server
- **Embedded Lua backend**: The UI server can run a Lua backend from the `lua/` subdirectory of the embedded app or the directory supplied with `--dir`

## Frontend Webapp Hosting

The UI server functions as a web server, hosting the frontend webapp. By default, the frontend is embedded as a compressed archive within the binary.

**Default mode (embedded site):**
- Serves directly from the bundled, compressed site without extraction
- Zero-configuration deployment - single binary includes everything

**Custom site mode (`--dir` flag):**
- Serves from a specified directory instead of the embedded site
- Allows users to customize or replace the frontend entirely

**Site management subcommands:**
- `extract` - Extract the bundled site to the filesystem for customization
- `bundle` - Create a new binary with a custom site bundled in
- `ls` - List files in the bundled site
- `cat` - Display contents of a bundled file
- `cp` - Copy files from the bundled site

## Configuration

The UI server reads configuration from an optional `config.toml` file:

**Configuration sources (in priority order):**
1. Command-line flags (highest priority)
2. Environment variables
3. `config.toml` in `--dir` directory or embedded storage
4. Built-in defaults (lowest priority)

### Configuration Options

| Option          | CLI Flag            | Env Var              | TOML Path         | Default              | Description                      |
|-----------------|---------------------|----------------------|-------------------|----------------------|----------------------------------|
| Host            | `--host`            | `UI_HOST`            | `server.host`     | `"0.0.0.0"`          | Browser listen address           |
| Port            | `--port`            | `UI_PORT`            | `server.port`     | `8080`               | Browser listen port              |
| Socket          | `--socket`          | `UI_SOCKET`          | `server.socket`   | (see below)          | Backend API socket               |
| Site directory  | `--dir`             | `UI_DIR`             | -                 | (embedded)           | Custom site directory            |
| Storage type    | `--storage`         | `UI_STORAGE`         | `storage.type`    | `"memory"`           | `memory`, `sqlite`, `postgresql` |
| Storage path    | `--storage-path`    | `UI_STORAGE_PATH`    | `storage.path`    | `"ui.db"`            | SQLite file path                 |
| Storage URL     | `--storage-url`     | `UI_STORAGE_URL`     | `storage.url`     | -                    | PostgreSQL connection URL        |
| Lua enabled     | `--lua`             | `UI_LUA`             | `lua.enabled`     | `true`               | Enable Lua backend               |
| Lua path        | `--lua-path`        | `UI_LUA_PATH`        | `lua.path`        | `"lua/"`             | Lua scripts directory            |
| Session timeout    | `--session-timeout`    | `UI_SESSION_TIMEOUT`    | `session.timeout`    | `"24h"`  | Session expiration (`0` = never)       |
| Connection timeout | `--connection-timeout` | `UI_CONNECTION_TIMEOUT` | `session.connection` | `"5s"`   | Grace period for frontend reconnection |
| Log level          | `--log-level`          | `UI_LOG_LEVEL`          | `logging.level`      | `"info"` | `debug`, `info`, `warn`, `error`       |

### Command-Line Usage

```
ui [command] [flags]

Server Commands:
  serve       Start the UI server (default)

Site Management Commands:
  extract     Extract bundled site to filesystem
  bundle      Create binary with custom site bundled
  ls          List files in bundled site
  cat         Display contents of a bundled file
  cp          Copy files from bundled site

Protocol Commands (connect to running server via socket):
  create      Create a variable
  destroy     Destroy a variable
  update      Update a variable's value and/or properties
  watch       Watch a variable for changes
  unwatch     Stop watching a variable
  get         Get variable values from storage
  poll        Get pending responses (with optional long-polling)

Server Flags:
  --host string              Browser listen address (default "0.0.0.0")
  --port int                 Browser listen port (default 8080)
  --socket string            Backend API socket path (default "/tmp/ui.sock")
  --dir string               Serve from directory instead of embedded site
  --storage string           Storage type: memory, sqlite, postgresql (default "memory")
  --storage-path string      SQLite database path (default "ui.db")
  --storage-url string       PostgreSQL connection URL
  --lua                      Enable Lua backend (default true)
  --lua-path string          Lua scripts directory (default "lua/")
  --session-timeout duration Session expiration (default 24h, 0=never)
  --log-level string         Log level: debug, info, warn, error (default "info")

Protocol Flags:
  --socket string            Connect to server socket (default: platform-specific)

Global Flags:
  --help                     Show help
```

### Backend Socket

The backend API uses a local socket for communication between the UI server and backend programs:

| Platform | Socket Type        | Default Path              |
|----------|--------------------|---------------------------|
| POSIX    | Unix domain socket | `/tmp/ui.sock`            |
| Windows  | Named pipe         | `\\.\pipe\ui`             |

The socket path can be customized via `--socket`, `UI_SOCKET`, or `server.socket` in config.

### Backend Protocol Detection

The socket accepts two protocols, auto-detected from the first bytes of each connection:

| Protocol | Detection                                           | Use Case                          |
|----------|-----------------------------------------------------|-----------------------------------|
| Packet   | First 4 bytes are binary length                     | Native backends (Go, Lua, etc.)   |
| HTTP     | First bytes are ASCII method (`GET `, `POST`, etc.) | REST clients, curl, shell scripts |

**Packet protocol format:**
```
[4-byte big-endian length][JSON payload]
```

Each message is a length-prefixed JSON object. This is efficient for persistent connections where backends send many messages.

**HTTP protocol:**
Standard HTTP/1.1 REST API. The server delegates to its HTTP handler when it detects an HTTP method.

**Detection logic:**
```
peek 4 bytes
if bytes match /^(GET |POST|PUT |DELE|HEAD|PATC|OPTI)/ → HTTP
else → interpret as 4-byte length, read packet
```

This allows the same socket to serve both native packet-based backends and HTTP REST clients without configuration.

### Protocol Commands

Protocol commands connect to a running UI server via the named pipe and forward protocol operations.

**Response model:** Every REST call / CLI command returns any pending responses (updates, errors, etc.) accumulated since the last call. This allows push-based protocol messages to be delivered to polling clients.

```bash
# Create a variable with parent ID 1
ui create --parent 1 --value '{"name": "Alice"}' --props 'type=Person'
# Returns: {"id": 5, "pending": [...]}

# Update a variable
ui update --id 5 --value '{"name": "Bob"}'
ui update --id 5 --props 'inactive='   # unset a property

# Get variable values
ui get 1 2 3

# Watch/unwatch
ui watch --id 1
ui unwatch --id 1

# Destroy a variable
ui destroy --id 5

# Poll for pending responses without sending a command
ui poll
ui poll --wait 30s   # long-poll with timeout
```

**Pending responses** include:
- `update` messages from watched variables
- `error` messages from failed operations
- `destroy` notifications for destroyed variables

The `poll` command (and REST equivalent) retrieves pending responses without performing any protocol operation. Use `--wait` for long-polling to block until responses are available or timeout expires.

These commands enable shell scripts and other programs to interact with the UI server without implementing the full protocol.

### Example `config.toml`

```toml
[server]
host = "0.0.0.0"
port = 8080
socket = "/tmp/ui.sock"   # backend API socket

[storage]
type = "sqlite"           # "memory", "sqlite", "postgresql"
path = "data/ui.db"       # for sqlite
# url = "postgres://..."  # for postgresql

[lua]
enabled = true
path = "lua/"             # relative to --dir or embedded root

[session]
timeout = "24h"           # session expiration (0 = never)
connection = "5s"         # grace period for frontend reconnection

[logging]
level = "info"            # "debug", "info", "warn", "error"
```

### Loading Behavior

- Embedded mode: Reads `config.toml` from the bundled archive if present
- Custom site mode (`--dir`): Reads `config.toml` from the specified directory
- Missing config file: Uses defaults (memory storage, port 8080, Lua enabled)

## Storage Options

- **Memory**: In-memory only, lost on restart (simplest, for development)
- **SQLite**: Persistent local database
- **PostgreSQL**: Production-grade persistent storage

## Technology Stack

- **Backend**: Go
- **Frontend**: HTML + Shoelace web components
- **Communication**: WebSocket (primary), HTTP/REST
- **Storage**: Memory / SQLite / PostgreSQL
