# MCPServer

**Source Spec:** interfaces.md

## Responsibilities

### Knows
- uiServer: Reference to UI server instance
- resources: List of available MCP resources
- tools: List of available MCP tools
- activeSession: Current session for AI interaction

### Does
- initialize: Set up MCP server connection
- listResources: Return available resources (presenter types, viewdefs, session state)
- listTools: Return available tools
- handleResourceRequest: Process resource queries
- handleToolCall: Execute tool operations
- sendNotification: Push events to AI client
- shutdown: Clean up MCP connection

## Collaborators

- MCPResource: Individual resource handlers
- MCPTool: Individual tool handlers
- SessionManager: Session operations
- LuaRuntime: Lua code execution

## Sequences

- seq-mcp-create-session.md: AI creating session
- seq-mcp-create-presenter.md: AI creating presenter
- seq-mcp-receive-event.md: AI receiving user events
