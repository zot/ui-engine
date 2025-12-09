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

## Storage Options

- **Memory**: In-memory only, lost on restart (simplest, for development)
- **SQLite**: Persistent local database
- **PostgreSQL**: Production-grade persistent storage

## Technology Stack

- **Backend**: Go
- **Frontend**: HTML + Shoelace web components
- **Communication**: WebSocket (primary), HTTP/REST
- **Storage**: Memory / SQLite / PostgreSQL
