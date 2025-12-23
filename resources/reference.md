# UI Platform Reference

Welcome to the UI Platform. This reference guide provides the essential knowledge needed to build and manage interactive user interfaces using Lua and View Definitions (viewdefs).

## Core Concepts

The platform follows a **Server-Side UI** architecture where the application state and logic reside in a Lua session on the server, and the frontend (browser) acts as a thin renderer.

### 1. The App Object (Logical Root)
Every session has a logical root object, typically named `app`.
- All data visible on the screen should be reachable from this object.
- The AI agent interacts with this object via the `mcp` global and `ui_run` tool.
- **State Inspection:** Use `ui://state` to see the current JSON representation of the logical root.

### 2. Presenters and Domain Objects
- **Domain Objects:** Pure data tables (e.g., a `Contact` or `Task`).
- **Presenters:** Tables that wrap domain objects and add UI-specific state (e.g., `isEditing`) and behaviors (methods like `save()` or `delete()`).
- **Pattern:** Keep data clean in domain objects; put interaction logic in presenters.

### 3. View Definitions (Viewdefs)
Viewdefs are HTML templates that define how a Lua object type should be rendered.
- **Bindings:** Use `ui-value`, `ui-action`, and other attributes to link HTML elements to Lua properties and methods.
- **Path Resolution:** All paths are resolved on the server relative to the object being rendered.

## Detailed Guides

- [Viewdef Syntax](ui://viewdefs) - Complete guide to `ui-*` attributes and path syntax.
- [Lua API & Patterns](ui://lua) - How to define classes, handle state, and use the `session` and `mcp` globals.
- [AI Interaction Guide](ui://mcp) - Best practices for agents to build "tiny apps" and handle user collaboration.

## Quick Start for Agents

1. **Configure & Start:** Use `ui_configure` then `ui_start`.
2. **Define Logic:** Use `ui_run` to define your Lua classes.
3. **Define UI:** Use `ui_upload_viewdef` to upload your HTML templates.
4. **Show UI:** Instantiate your app class and assign it to `mcp.state`.
5. **Open Browser:** Use `ui_open_browser` to show the result to the user.
6. **Iterate:** Listen for notifications via `mcp.notify` and update state or viewdefs on the fly.
