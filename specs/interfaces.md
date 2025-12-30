# Interfaces

## Frontend Server

The webapp that runs in the browser is a custom "frontend server" that hosts the remote user interface controlled by the backend.

**Session URLs:**
- Connecting to `http://SITE` redirects to `http://SITE/NEW-SESSION-ID`
- Each session has a unique ID embedded in the URL path
- Session URLs can be shared or bookmarked for reconnection

**Shared Worker Architecture:**
- A SharedWorker maintains the concept of a "main" tab connected to the backend
- The main tab holds the primary WebSocket connection
- Other tabs coordinate through the SharedWorker
- Connecting to `http://SITE/SESSION-ID` when a tab is already connected:
  - Activates the existing connected tab (via desktop notification)
  - Closes the new tab

**SPA History Management:**
- Each session uses SPA-style history management
- Objects (presenters) can be registered with URL paths by the backend app (after the session ID)
  - Only presenters explicitly registered by the backend are addressable via URL
- Navigation updates the URL without full page reloads
- Back/forward navigation restores presenter state

**Tab Activation:**

Opening a new browser tab/window to a session URL:
- `http://SITE/SESSION-ID` - Activates the existing connected tab
  - Sends a "click to focus" desktop notification
  - Clicking the notification brings the main tab to focus
  - If history length == 1, closes the new tab; otherwise goes back in history
- `http://SITE/SESSION-ID/PATH` - Activates and navigates to PATH
  - Backend can open this URL to direct the user to a specific page
  - Same activation behavior as above
- `http://SITE/SESSION-ID` when no session exists - Shows an error page
  - No desktop notification is sent
  - The tab remains open (not auto-closed)

Enables backends/AIs to bring the UI to the user's attention by opening the session URL.

**Reconnection:**
- Frontend can reconnect to any session that hasn't timed out
- Session timeout is configured via `--session-timeout` (default: 24h, 0=never)
- Allows page refreshes, network interruptions, and browser restarts without losing session state
- Session state is preserved until session timeout expires

**Browser Communication:**
- **WebSocket**: Real-time bidirectional communication (via main tab)
- **JSONP**: For legacy/cross-origin scenarios

## Backend Integration Patterns

- **REST API (HTTP)**: Standard request/response
- **WebSocket**: Persistent real-time connection
- **MCP (Model Context Protocol)**: For AI assistant integration (see ui-mcp project)
- **Command Line**: Mirrors the REST API for simple shell script integration
- **Embedded Lua**: Backend logic in the `lua/` subdirectory runs within the UI server process

## Backend Modes

The UI server supports three backend configurations:

**1. Embedded Lua only (`--lua`, no connected backend):**
- Complete app runs in embedded Lua
- `main.lua` creates variable 1 and handles all logic
- Best for: Simple apps, demos, prototypes

**2. Connected backend only (no `--lua`):**
- Complete app runs in external backend (Go, etc.) connected via socket
- Backend creates variable 1 and handles all logic
- Best for: Apps that need full backend language capabilities

**3. Hybrid (`--lua` + connected backend):**
- Both embedded Lua and external backend active
- Developer decides where variable 1 is created
- Allows embedded Lua for reusable UI behavior with backend as "plugin"
- Best for: Complex apps where Lua handles common patterns and backend handles app-specific logic

## Backend Socket

External backends connect to the UI server via Unix socket (or named pipe on Windows).

**Protocol:**
- Session-wrapped batches: `{"session": "abc123", "messages": [...]}`
- When a batch arrives with a new session ID, the backend creates a corresponding session
- Backend is responsible for creating variable 1 (unless hybrid mode with Lua creating it)

**Default path:** `/tmp/ui.sock` (Unix) or `\\.\pipe\ui` (Windows)

## Embedded Lua Runtime

When `--lua` is enabled (default: true), the UI server provides an embedded Lua runtime for presentation logic.

**Session-Based Architecture:**
- Each frontend session has a corresponding Lua session
- When a frontend connects and creates a new session, the UI server:
  1. Creates a new Lua session with a `session` global
  2. Loads and executes `main.lua` from the site's `lua/` directory (if it exists)
- Executing `main.lua` serves as the notification of a new session
- In Lua-only mode, `main.lua` is responsible for:
  - Creating variable 1 (the app variable) with initial app state
  - Defining presenter objects with methods that handle `ui-action` calls
- In hybrid mode, `main.lua` may set up reusable behaviors while the backend creates variable 1

**Session Object:**
- A `session` global is available when `main.lua` executes
- Provides methods for variable management (see Lua Session API in libraries.md)

**Execution Model:**
- UI server creates a Lua executor goroutine with a channel for zero-arg functions
- All variable path sets and method calls execute through this channel
- Ensures single-threaded Lua access (Lua VMs are not thread-safe)
- Variable updates that trigger Lua methods are queued and processed sequentially

**Dynamic Code Loading:**
- After initial load, additional code can be loaded via the `lua` property on variable 1
- Two modes depending on the value format:
  1. **Inline code**: If the value is Lua code, evaluate it directly
  2. **File reference**: If the value ends with `.lua`, load from `<dir>/lua/<filename>`

- Examples via protocol:
  ```json
  // Inline code
  {"type": "update", "id": 1, "properties": {"lua": "ui.registerPresenter('MyApp', {...})"}}

  // File reference
  {"type": "update", "id": 1, "properties": {"lua": "helpers.lua"}}
  ```

**Lua API:**
- `ui.registerPresenter(name, table)` - Register a presenter type
- `ui.log([level,] message)` - Log from Lua code (delegates to `Config.Log`)
- `ui.json_encode(value)` / `ui.json_decode(string)` - JSON conversion

## Reliability

The server is designed to remain stable even when application logic fails:

- **Panic recovery**: Panics during message processing are caught and logged without crashing the server
- **Isolated failures**: A bad update from the frontend or a bug in backend Lua code affects only that operation, not the entire server
- **Session continuity**: Other sessions and subsequent operations continue normally after a recovered panic

This ensures that development errors, malformed client messages, or edge cases in application logic don't bring down the server.

## Known Limitations

### Single Connection Per Session

Currently, each session assumes a single WebSocket connection. Opening multiple browser tabs to the same session URL will cause undefined behavior:

- Both tabs will receive updates, but state synchronization may be inconsistent
- User actions from different tabs may interleave unpredictably
- The SharedWorker architecture (described above) is designed to prevent this by activating the existing tab

**Workaround:** The SharedWorker ensures only one tab is "active" per session. If a user opens a second tab to the same session, the first tab is activated and the second tab closes.

**Future consideration:** Supporting multiple connections per session would require:
- Consolidating per-connection and per-session state in WebSocketEndpoint
- Ensuring message ordering and state consistency across connections
- Defining behavior for conflicting concurrent actions
