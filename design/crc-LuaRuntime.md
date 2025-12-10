# LuaRuntime

**Source Spec:** interfaces.md, deployment.md

## Responsibilities

### Knows
- state: Lua VM state
- loadedModules: Map of loaded Lua modules
- presenterTypes: Registered Lua presenter types
- luaDir: Directory for Lua files (lua/ or --dir)

### Does
- initialize: Create Lua VM and load standard library
- loadFile: Load and execute Lua file
- loadCode: Load and execute Lua code string
- registerPresenterType: Register Lua class as presenter type
- callMethod: Invoke method on Lua presenter
- getPresenterValue: Get value from Lua presenter
- setPresenterValue: Set value on Lua presenter
- shutdown: Clean up Lua VM

## Collaborators

- LuaPresenterLogic: Presenter implementations
- MCPTool: Loads code via MCP
- ProtocolHandler: Invokes Lua methods on actions
- VariableStore: Binds Lua objects to variables

## Sequences

- seq-load-lua-code.md: Loading Lua presenter logic
- seq-lua-handle-action.md: Handling user actions in Lua
