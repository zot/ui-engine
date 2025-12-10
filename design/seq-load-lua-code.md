# Sequence: Load Lua Code

**Source Spec:** interfaces.md, deployment.md
**Use Case:** Loading Lua presenter logic into runtime

## Participants

- Source: MCP tool or file system
- LuaRuntime: Lua VM manager
- LuaPresenterLogic: Presenter type definition
- VariableStore: Variable storage

## Sequence

```
     Source             LuaRuntime        LuaPresenterLogic        VariableStore
        |                      |                      |                      |
        |---loadCode(lua)----->|                      |                      |
        |                      |                      |                      |
        |                      |---parse()----------->|                      |
        |                      |                      |                      |
        |                      |---execute()--------->|                      |
        |                      |                      |                      |
        |                      |     [Lua defines presenter type]            |
        |                      |                      |---defineType-------->|
        |                      |                      |   (typeName)         |
        |                      |                      |                      |
        |                      |                      |---defineMethod------>|
        |                      |                      |   (name, func)       |
        |                      |                      |                      |
        |                      |                      |---defineProperty---->|
        |                      |                      |   (name, get, set)   |
        |                      |                      |                      |
        |                      |<--type registered----|                      |
        |                      |                      |                      |
        |                      |---registerType------>|                      |
        |                      |   (for create prop)  |                      |
        |                      |                      |---register---------->|
        |                      |                      |                      |
        |<--success------------|                      |                      |
        |                      |                      |                      |
```

## Notes

- Lua code loaded from file (lua/ directory) or via MCP
- Code executed in Lua VM to define presenter types
- Types registered for use with `create` property
- Methods callable via ui-action bindings
- Properties accessible via ui-value bindings
- Lua objects can be bound to variables
