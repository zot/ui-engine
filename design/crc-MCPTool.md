# MCPTool

**Source Spec:** interfaces.md

## Responsibilities

### Knows
- name: Tool identifier
- description: Human-readable description
- inputSchema: JSON schema for tool parameters
- handler: Function to execute tool

### Does
- define: Define tool schema (name, description, input schema)
- handle: Execute tool logic (interface implementation)

### Standard Tools
- ui_get_state: Get current session state (Variable 1)
- ui_run: Execute Lua code in session context
- ui_upload_viewdef: Add dynamic view definition

## Collaborators

- MCPServer: Registers and invokes tools
- SessionManager: Session creation
- VariableStore: Presenter creation/update
- ViewdefStore: Viewdef management
- LuaRuntime: Lua code loading
- Router: URL path registration

## Sequences

- seq-mcp-create-session.md: Session creation tool
- seq-mcp-create-presenter.md: Presenter/viewdef creation
- seq-mcp-receive-event.md: Event handling tools
