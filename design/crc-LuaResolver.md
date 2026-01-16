# LuaResolver

**Source Spec:** libraries.md, protocol.md (Variable Wrappers section)

## Responsibilities

### Knows
- Session: Reference to owning LuaSession (provides Lua state, logging)

### Does
- Get: Navigate Lua tables/wrappers to retrieve values at path elements (string keys, integer indexes)
- Set: Assign values in Lua tables at path elements
- Call: Invoke zero-argument methods on Lua tables (for computed getters)
- CallWith: Invoke one-argument methods with a value (for computed setters)
- CreateWrapper: Create/reuse wrapper objects for variables with `wrapper` property
- CreateValue: Create value objects for variables with `create` property
- GetType: Determine a value's type from metatable or `type` field
- ConvertToValueJSON: Convert Lua values to JSON-compatible format (arrays to slices, objects to refs)

## Collaborators

- LuaSession: Provides Lua VM state, type checking (isArray), logging
- change-tracker.Tracker: Uses resolver for path navigation and wrapper creation
- change-tracker.Variable: Target of CreateWrapper/CreateValue operations
- ViewList: Go wrapper type resolved by Get
- ViewListItem: Go wrapper type resolved by Get

## Path Navigation

LuaResolver handles navigation through various object types:

**Lua Tables:**
- String path elements: `GetField(tbl, key)`
- Integer path elements: `RawGetInt(tbl, index+1)` (Lua is 1-indexed)

**Go Wrappers:**
- ViewList: Supports `items` property returning item array
- ViewListItem: Supports `item`, `index`, `list`, `type` properties
- Slices: Supports integer index access

**Method Calls:**
- `method()`: Zero-argument method call, returns result
- `method(_)`: One-argument method call (for read/write paths)

## Wrapper Creation

`CreateWrapper(variable)` handles wrapper lifecycle:

1. **Reuse check**: If `variable.WrapperValue` exists:
   - Update wrapper's `value` property
   - Call `sync()` if it exists (for ViewList sync)
   - Return existing wrapper

2. **Go registry**: Check `GetGlobalWrapperFactory(wrapperType)` for Go implementations

3. **Lua globals**: Look up `wrapperType` in Lua globals:
   - Must be a table with `new(self, variable)` method
   - Call `WrapperType:new(variable)` with LuaVariable wrapper
   - Store result in `variable.WrapperValue`
   - Set `type` property to wrapper type name

## Value Creation

`CreateValue(variable, type, value)` creates presenter instances:

1. Check Go registry via `GetGlobalCreateFactory(type)`
2. Look up type in Lua globals
3. Call `Type:new(self, value)` constructor
4. Return created instance

## Type Resolution

`GetType(variable, obj)` determines type for viewdef lookup:

1. Check metatable for `type` field
2. Fall back to direct `type` field on table
3. For Go objects, check registered factory types
4. Return empty string if no type found

## Value Conversion

`luaValueToGo` converts Lua values to Go:
- `LBool` -> `bool`
- `LNumber` -> `float64`
- `LString` -> `string`
- `*LTable` -> kept as-is for navigation
- `LNil` -> `nil`

`ConvertToValueJSON` handles serialization:
- Array tables: Convert to `[]any` with recursive element conversion
- Object tables: Return as-is for tracker to register as object refs

## Sequences

- seq-lua-resolve.md: Path resolution through Lua objects
- seq-wrapper-transform.md: Wrapper creation and reuse

## Notes

### Interface Implementation

LuaResolver implements `changetracker.Resolver` interface with all required methods. This allows the change-tracker to work with Lua values transparently.

### LuaVariable Wrapper

When calling Lua wrapper constructors, a LuaVariable wrapper table is created with:
- `_id`: Variable ID number
- `getValue()`: Returns current variable value
- `getProperty(name)`: Returns variable property
- `getWrapper()`: Returns existing wrapper or nil

### Thread Safety

All Lua operations go through the session's executor channel, ensuring thread-safe access to the Lua state.
