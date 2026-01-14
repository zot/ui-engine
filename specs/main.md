# UI Platform Specification

UI is a platform that provides an API for making, displaying, and interacting with remote user interfaces from programs and AIs.

## Basic Premise

The platform implements a **UI server** that connects static frontend code (a "presentation service") to pluggable backends:

- **The UI can:**
  - Access and change state in presentation and domain objects
  - Send messages to presentation and domain objects

- **Backends** connect to the UI server via:
  - A service port (WebSocket/HTTP)
  - The UI server command line as a proxy

- **Presentation and domain objects** can exist in:
  - A backend (external program)
  - The UI server itself (via embedded Lua)
  - Both (hybrid model)

## UI Server Architecture

The UI server has two conceptual layers:

### Frontend Layer (Session Management)

The frontend layer manages browser connections and sessions:
- **Session**: Represents a user's session with the UI server
  - Tracks connected browser tabs (multiple tabs can share one session)
  - Routes messages to the appropriate backend
  - Lightweight - does not manage variables directly
- **SessionManager**: Creates/destroys sessions, manages session lifecycle

### Backend Layer (Variable Management)

Each session has an associated backend that handles variable management. All protocol messages (create, update, watch, etc.) are forwarded to the backend within a session-identifying batch wrapper.

**Hosted Backend (Lua)**
- Runs embedded Lua code per session
- Owns a `change-tracker.Tracker` for variable management
- Manages watch subscriptions (which connections watch which variables)
- Detects changes and sends updates to watching connections
- Variable IDs are scoped to the session (each session has its own variable 1)

**Proxied Backend**
- Relays all messages to/from an external backend program
- Messages are wrapped with session ID for routing
- External backend is source of truth for variables

This separation allows the UI server to support both:
- Self-contained Lua applications (hosted)
- External backend integration (proxied)

## Design Principles

### Frictionless Development

The platform is designed for **frictionless UI development**. Developers should be able to create UIs with minimal boilerplate, configuration, or registration ceremonies:

- **Convention over configuration**: Features "just work" through sensible defaults and auto-discovery
- **Zero registration**: Define a wrapper type and use it by name - no explicit registration calls required
  - **Lua**: Define a global table with `computeValue` method, use it by name
  - **Go**: Use `init()` with `RegisterWrapperType()` for automatic registration at import
- **Declarative binding**: HTML attributes like `ui-value="path"` automatically bind to backend data
- **Auto-discovery of wrappers**: Use `wrapper=MyWrapper` in a path and the platform finds it automatically
- **Minimal backend code**: The app variable is the only required setup point

### Centralized Logging

To ensure consistent output and granular control over debug information:
- **All logging must go through the `Config` object.** 
- Subsystems delegate logging to the central `Config.Log` method.
- Verbosity levels (0-4) are managed centrally, eliminating redundant flags in subsystems.

## Target Users

- **AI Assistants**: AIs like Claude that need to present rich UIs to users during conversations
- **Backend Services**: Programs and services that need a UI without bundling a frontend

## Specification Outline

### [Core Concepts](#core-concepts)
- Sessions, Presenters, Views, Hot-Loading System, Standard Presenters

### [Variable Protocol](protocol.md)
- Variable Identity, Values, Properties
- Protocol Messages (create, destroy, update, watch)
- Source of truth responsibilities
- Watch tallying

### [Viewdef Binding](viewdefs.md)
- View Definitions and delivery
- Value bindings (`ui-value`, `ui-attr-*`, `ui-class-*`, `ui-style-*`)
- Event bindings (`ui-event-*`)
- Backend paths

### [Deployment](deployment.md)
- Deployment modes (standalone, FastCGI, CLI, embedded Lua)
- Frontend webapp hosting
- Technology stack

### [Interfaces](interfaces.md)
- Session URLs (`SITE/SESSION-ID`)
- Frontend server (SharedWorker, SPA history, tab activation)
- Backend integration patterns (REST, WebSocket, MCP, CLI, Lua)
- **MCP Server** - Primary AI integration point

### [Backend Data Models](data-models.md)
- Unbound model (UI server as database)
- Bound model (external data integration)
- Hybrid usage

### [Libraries](libraries.md)
- Backend library for Go and Lua (connection, path navigation, change detection)
- Frontend library (SPA navigation, view rendering, widgets)

### [UI Components](components.md)
- Basic form elements
- Rich content
- Advanced widgets

### [Demo Application](demo.md)
- Contact Manager example with Lua backend
- CRUD operations

---

## Core Concepts

### Sessions

- Each user connection gets a unique session ID
- Sessions are anonymous (no authentication required)
- Sessions are isolated - single user per session, no sharing between sessions

### Presenters

Presenters are JSON objects that represent UI state on the backend.

- Each presenter has a **type** (e.g., "form", "table", "chart")
- The backend manages a **tree of variables** that hold state and reach into presenter objects

### Views

Views are HTML snippets associated with presenter types and bound to data within them.

- Each presenter type can have multiple views, identified by a **view name**
- Default view name is `"DEFAULT"`
- Views contain arbitrary HTML/Shoelace widgets
- The view + backend together manage the complete UI state of the browser page

### Hot-Loading System

When `--hotload` is enabled, the server watches directories for file changes and automatically reloads modified files. This is a unified system that handles both Lua scripts and viewdefs.

**Watched directories:**
- Lua scripts: `lua/` (or configured lua path)
- Viewdefs: `html/viewdefs/`

**Symlink tracking:** When a watched file is a symlink, the server also watches the target directory. Changes to either the symlink or the target file trigger a reload. When symlinks are added, modified, or removed, watched directories update accordingly. This supports development workflows where files are symlinked from another location.

**Backend reload behavior:**
- Only reloads files that have already been loaded (ignores new files until explicitly required)
- Debounces rapid file changes to avoid multiple reloads
- After reload, triggers session refresh to push changes to connected clients

**Lua-specific behavior:**
- Re-executes modified files in each active session
- Sets `session.reloading = true` before reload, `false` after
- Sessions maintain state between reloads (see conventions below)
- Module load tracking handles circular dependencies safely

**Viewdef-specific behavior:**
- Reloads file content and updates the viewdefs map
- For each session that received the changed viewdef, queues a re-push via variable 1's `viewdefs` property
- Frontend finds views with matching `data-ui-viewdef` attribute and re-renders them

**Hot-loading conventions (Lua):**

For hot-loading to preserve state, Lua code should follow these conventions:

1. **Use session:prototype()** - prototypes are stored in session registry and preserved across reloads:
   ```lua
   MyApp = session:prototype("MyApp", {
       title = "My Application"
   })
   ```

2. **Check for existing app** - avoid recreating variable 1:
   ```lua
   if not session:getApp() then
       session:createAppVariable(MyApp:new())
   end
   ```

3. **Instance mutation** - define a `mutate()` method on the prototype for schema migrations

See `USAGE.md` for complete hot-loading documentation.

### Standard Presenters

Built-in presenter types are provided in the backend library.

**App Presenter:**
- `currentPage()` - Returns `history[historyIndex]`
- `url` - Current URL
- `historyIndex` - Current position in history
- `history` - Array of page objects (set by backend)

**List Presenter:**
- `items` - The objects in the list
- `selectionIndex` - Index of selected item
- `disabled` - Defaults to false
