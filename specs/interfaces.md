# Interfaces

## Frontend Server

The webapp that runs in the browser is a custom "frontend server" that hosts the remote user interface controlled by the backend.

**Shared Worker Architecture:**
- A SharedWorker maintains the concept of a "main" tab connected to the backend
- The main tab holds the primary WebSocket connection
- Other tabs coordinate through the SharedWorker

**SPA History Management:**
- Each session uses SPA-style history management
- Objects (presenters) are registered with URL paths on the backend
- Navigation updates the URL without full page reloads
- Back/forward navigation restores presenter state

**Tab Activation:**
- URL with `?activate=SESSION-ID` triggers tab focus
- The main tab sends a "click to focus" desktop notification
- Clicking the notification navigates the user to the main tab
- Enables backends/AIs to bring the UI to the user's attention

**Browser Communication:**
- **WebSocket**: Real-time bidirectional communication (via main tab)
- **JSONP**: For legacy/cross-origin scenarios

## Backend Integration Patterns

- **REST API (HTTP)**: Standard request/response
- **WebSocket**: Persistent real-time connection
- **MCP (Model Context Protocol)**: For AI assistant integration
- **Command Line**: Mirrors the REST API for simple shell script integration
- **Embedded Lua**: Backend logic in the `lua/` subdirectory runs within the UI server process
