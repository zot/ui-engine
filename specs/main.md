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

## Target Users

- **AI Assistants**: AIs like Claude that need to present rich UIs to users during conversations
- **Backend Services**: Programs and services that need a UI without bundling a frontend

## Specification Outline

### [Core Concepts](#core-concepts)
- Sessions, Presenters, Views, Standard Presenters

### [Variable Protocol](protocol.md)
- Variable Identity, Values, Properties
- Protocol Messages (create, destroy, update, watch)
- Source of truth responsibilities
- Watch tallying

### [Viewdef Binding](viewdefs.md)
- View Definitions and delivery
- Value bindings (`ui-value`, `ui-attr-*`, `ui-class-*`, `ui-style-*-*`)
- Event bindings (`ui-event-*`)
- Backend paths

### [Deployment](deployment.md)
- Deployment modes (standalone, FastCGI, CLI, embedded Lua)
- Frontend webapp hosting
- Storage options (Memory, SQLite, PostgreSQL)
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
- In-memory storage, CRUD operations

---

## Core Concepts

### Sessions

- Each user connection gets a unique session ID
- Sessions are anonymous (no authentication required)
- Sessions are isolated - single user per session, no sharing between sessions

### Presenters

Presenters are JSON objects that represent UI state on the backend.

- Each presenter has a **type** (e.g., "form", "table", "chart")
- Presenters are stored in-memory, SQLite, or PostgreSQL
- The backend manages a **tree of variables** that hold state and reach into presenter objects

### Views

Views are HTML snippets associated with presenter types and bound to data within them.

- Each presenter type can have multiple views, identified by a **view name**
- Default view name is `"DEFAULT"`
- Views contain arbitrary HTML/Shoelace widgets
- The view + backend together manage the complete UI state of the browser page

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
