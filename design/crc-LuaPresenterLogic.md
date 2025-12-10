# LuaPresenterLogic

**Source Spec:** interfaces.md

## Responsibilities

### Knows
- typeName: Presenter type name
- methods: Map of method name to Lua function
- properties: Map of property name to getter/setter

### Does
- defineType: Create new presenter type in Lua
- defineMethod: Add method to presenter type
- defineProperty: Add property with getter/setter
- instantiate: Create instance of presenter type
- handleAction: Process ui-action calls
- updateProperty: Handle property changes from frontend
- notifyChange: Signal that presenter state changed

## Collaborators

- LuaRuntime: Executes Lua code
- Presenter: Base presenter interface
- ProtocolHandler: Receives action triggers
- WatchManager: Notifies of presenter changes

## Sequences

- seq-load-lua-code.md: Defining presenter types
- seq-lua-handle-action.md: Action handling flow
- seq-mcp-create-presenter.md: AI creating Lua presenters
