# Architecture

```
[browser] <--WebSocket--> [UI Server] <--> [backend program(s) / MCP]

[browser] <--HTTP--> [web server] <--FastCGI--> [UI Instance] <--> [UI Server]
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

### Bundle Format

Sites are bundled by appending a ZIP archive to the end of the binary, with a footer that identifies the bundle:

```
[executable binary][ZIP archive][footer]

Footer (24 bytes):
  [8 bytes: offset to ZIP start (little-endian int64)]
  [8 bytes: ZIP size (little-endian int64)]
  [8 bytes: magic marker "UISERVER"]
```

**Key properties:**
- **No recompilation required**: Bundling happens post-compilation by appending data
- **Cross-platform**: Pre-built binaries for all platforms can be bundled with any site
- **Re-bundleable**: A bundled binary can be re-bundled with a different site
- **Efficient**: ZIP data is read directly from the binary without extraction

**Detection:** At startup, the server reads the last 24 bytes. If the magic marker matches, the ZIP is served directly. Otherwise, the binary is unbundled.

**Bundle command:**
```bash
# Bundle a site into a new binary
ui bundle my-site -o my-app

# Re-bundle with different site (strips old bundle first)
./my-app bundle other-site -o other-app
```

### Site Directory Structure

Both embedded bundles and `--dir` directories use the same structure:

```
site/
├── html/           # Web files served to browsers (required)
│   ├── index.html  # Main entry point
│   ├── main.js     # Application JavaScript
│   └── ...         # Other web assets
├── config/         # Configuration files (optional)
│   └── config.toml # Server configuration
└── lua/            # Lua presentation code (optional)
    └── *.lua       # Lua scripts loaded at startup
```

**Directory purposes:**

| Directory | Purpose | Notes |
|-----------|---------|-------|
| `html/` | Static web files | Served at root URL; `index.html` is the SPA entry point |
| `config/` | Configuration | `config.toml` is loaded if present; overrides defaults |
| `lua/` | Lua scripts | Loaded when Lua is enabled; provides presentation logic |

**Embedded mode (default):**
- Server reads from ZIP archive appended to binary
- All paths are relative to ZIP root
- Example: `html/index.html` in ZIP → served at `/index.html`

**Directory mode (`--dir`):**
- Server reads from filesystem at specified path
- Structure must match: `<dir>/html/`, `<dir>/config/`, `<dir>/lua/`
- Example: `--dir my-app` → serves from `my-app/html/`

**Minimal site:**
```
my-site/
└── html/
    └── index.html
```

**Full site with Lua backend:**
```
my-site/
├── html/
│   ├── index.html
│   ├── main.js
│   └── styles.css
├── config/
│   └── config.toml
└── lua/
    ├── init.lua
    └── handlers.lua
```

## Configuration

The UI server reads configuration from an optional `config.toml` file:

**Configuration sources (in priority order):**
1. Command-line flags (highest priority)
2. Environment variables
3. `config.toml` in `--dir` directory or embedded storage
4. Built-in defaults (lowest priority)

### Configuration Options

| Option          | CLI Flag            | Env Var              | TOML Path         | Default     | Description                      |
|-----------------|---------------------|----------------------|-------------------|-------------|----------------------------------|
| Host            | `--host`            | `UI_HOST`            | `server.host`     | `"0.0.0.0"` | Browser listen address           |
| Port            | `--port`            | `UI_PORT`            | `server.port`     | `8080`      | Browser listen port              |
| Socket          | `--socket`          | `UI_SOCKET`          | `server.socket`   | (see below) | Backend API socket               |
| Site directory  | `--dir`             | `UI_DIR`             | -                 | (embedded)  | Custom site directory            |
| Lua enabled     | `--lua`             | `UI_LUA`             | `lua.enabled`     | `true`      | Enable Lua backend               |
| Lua path        | `--lua-path`        | `UI_LUA_PATH`        | `lua.path`        | `"lua/"`    | Lua scripts directory            |
| Lua hotload     | `--hotload`         | `UI_HOTLOAD`         | `lua.hotload`     | `false`     | Watch lua directory for changes  |
| Session timeout | `--session-timeout` | `UI_SESSION_TIMEOUT` | `session.timeout` | `"24h"`     | Session expiration (`0` = never) |
| Log level       | `--log-level`       | `UI_LOG_LEVEL`       | `logging.level`   | `"info"`    | `debug`, `info`, `warn`, `error` |
| Verbosity       | `-v` to `-vvvv`     | `UI_VERBOSITY`       | `logging.verbosity` | `0`        | Debug output level (0-4)         |

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
  get         Get variable values
  poll        Get pending responses (with optional long-polling)

Server Flags:
  --host string              Browser listen address (default "0.0.0.0")
  --port int                 Browser listen port (default 8080)
  --socket string            Backend API socket path (default "/tmp/ui.sock")
  --dir string               Serve from directory instead of embedded site
  --lua                      Enable Lua backend (default true)
  --lua-path string          Lua scripts directory (default "lua/")
  --hotload                  Watch lua directory for changes (default false)
  --session-timeout duration Session expiration (default 24h, 0=never)
  --log-level string         Log level: debug, info, warn, error (default "info")
  -v                         Verbosity level 1: connection events
  -vv                        Verbosity level 2: + protocol messages
  -vvv                       Verbosity level 3: + variable operations
  -vvvv                      Verbosity level 4: + variable values

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

### Verbosity Levels

The verbosity flag (`-v`) controls debug output for troubleshooting. Each level includes all output from lower levels. 

**Centralized Logging**: To ensure consistent formatting and control, all components must use the `Config.Log(level, format, ...)` method for output instead of direct library calls.

| Level | Flag    | Output                                           |
|-------|---------|--------------------------------------------------|
| 0     | (none)  | Normal operation, errors only                    |
| 1     | `-v`    | Connection events (connect, disconnect, reconnect) |
| 2     | `-vv`   | Protocol messages (create, update, watch, etc.)  |
| 3     | `-vvv`  | Variable operations (CRUD details, property changes) |
| 4     | `-vvvv` | Variable values (full JSON values on set/update) |

**Examples:**
```bash
# See connection activity
ui serve -v

# Debug protocol messages
ui serve -vv

# Full variable operation tracing
ui serve -vvv

# Show variable values (verbose, large output)
ui serve -vvvv
```

**Environment variable:** Set `UI_VERBOSITY=1`, `2`, `3`, or `4` for equivalent levels.

**TOML configuration:**
```toml
[logging]
verbosity = 2  # equivalent to -vv
```

### Example `config.toml`

```toml
[server]
host = "0.0.0.0"
port = 8080
socket = "/tmp/ui.sock"   # backend API socket

[lua]
enabled = true
path = "lua/"             # relative to --dir or embedded root
hotload = false           # watch for file changes

[session]
timeout = "24h"           # session expiration (0 = never)

[logging]
level = "info"            # "debug", "info", "warn", "error"
verbosity = 0             # 0=none, 1=connections, 2=messages, 3=variables
```

### Lua Hot-Loading

When `--hotload` is enabled, the server watches the lua directory for file changes and automatically reloads modified files for all active sessions.

**Watch behavior:**
- Watches the configured lua path (default: `lua/`)
- On file change, re-executes the modified file in each active session
- Sessions maintain state between reloads (see conventions below)

**Symlink handling:**
- If a file in the lua directory is a symlink, the server also watches the real (target) directory
- This supports development workflows where lua files are symlinked from another location
- Changes to either the symlink or the target file trigger a reload
- When a symlink is added, modified, or removed, the watched directories are updated accordingly

**Example with symlinks:**
```
lua/
├── main.lua              # regular file - watched directly
├── app.lua -> ../apps/myapp/app.lua   # symlink - also watches ../apps/myapp/
└── utils.lua -> /shared/utils.lua     # symlink - also watches /shared/
```

**Session refresh after hotload:**

After reloading Lua code, the server triggers a session refresh by executing an empty function via `ws.ExecuteInSession`. This causes the session's `AfterBatch` to run, which detects and pushes any viewdef or variable changes to the browser. The execution is wrapped in panic recovery to prevent a misbehaving Lua script from crashing the server - panics are logged as errors instead.

**Hot-loading conventions:**

For hot-loading to preserve state, Lua code should follow these conventions:

1. **Conditional prototype assignment** - preserve existing prototypes:
   ```lua
   MyApp = MyApp or {type = "MyApp"}
   MyApp.__index = MyApp
   ```

2. **Check for existing app** - avoid recreating variable 1:
   ```lua
   if not session:getApp() then
       session:createAppVariable(MyApp:new())
   end
   ```

3. **Instance mutation** - use `session:newVersion()` and `session:needsMutation()` for schema migrations

See `USAGE.md` for complete hot-loading documentation.

### Loading Behavior

- Embedded mode: Reads `config.toml` from the bundled archive if present
- Custom site mode (`--dir`): Reads `config.toml` from the specified directory
- Missing config file: Uses defaults (memory storage, port 8080, Lua enabled)

## Technology Stack

- **Backend**: Go
- **Frontend**: HTML + Shoelace web components
- **Communication**: WebSocket (primary), HTTP/REST
