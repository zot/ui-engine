# MCPTool

**Source Spec:** interfaces.md

## Responsibilities

### Knows
- name: Tool identifier
- description: Human-readable description
- inputSchema: JSON schema for tool parameters
- handler: Function to execute tool

### Does
- createSession: Create new session, return session URL
- createPresenter: Create presenter with type and properties
- updatePresenter: Update presenter properties or call method
- destroyPresenter: Remove presenter
- createViewdef: Create HTML viewdef for TYPE.VIEW
- updateViewdef: Modify existing viewdef
- loadPresenterLogic: Load Lua code into runtime
- registerUrlPath: Associate URL path with presenter
- activateTab: Bring user's browser tab to focus

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
