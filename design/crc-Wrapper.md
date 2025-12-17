# Wrapper

**Source Spec:** protocol.md, libraries.md

## Responsibilities

### Knows
- variable: The Variable object (received in constructor, stored for later access)
- value: The variable's value (from `variable:getValue()`, stored for convenience)
- managedObjects: Objects created and managed by this wrapper (e.g., ViewListItems)

### Does
- new(variable): Constructor receives Variable object, returns new or existing wrapper
- getWrapper: Check for existing wrapper via `variable:getWrapper()` for reuse pattern
- sync: Update internal state when value changes (on wrapper reuse)
- destroy: Clean up all managed objects when variable destroyed

## Collaborators

- Variable: Stores wrapper instance internally, provides getValue() and getWrapper()
- Resolver: Calls CreateWrapper(variable) whenever variable value changes
- ObjectRegistry: Registers wrapper object for child path navigation
- LuaRuntime: Hosts wrapper implementation (for embedded Lua)

## Notes

### Wrapper Behavior

The wrapper object itself **stands in for the variable's value** when child variables navigate paths. The wrapper is registered in the object registry and becomes the variable's navigation value. There is no `computeValue()` method - the wrapper IS the value.

### Wrapper Interface (Lua Convention)

```lua
Wrapper {
    new(variable)           -- Receives Variable, returns new or existing wrapper
    variable                -- Stored Variable object for later access
    value                   -- Stored value (from variable:getValue()) for convenience
}
```

The constructor receives the Variable object, allowing it to:
- Access variable properties (e.g., `item=ContactPresenter`)
- Store a reference for later use
- Check for existing wrapper via `variable:getWrapper()`
- Access the value via `variable:getValue()`

### Wrapper Creation and Reuse

`Resolver.CreateWrapper(variable)` is called **whenever the variable's value changes**. The wrapper can:

1. **Return existing wrapper** - Preserves internal state (selection, scroll position)
2. **Return new wrapper** - Creates fresh state
3. **Return nil** - No wrapper needed

This enables stateful wrappers like ViewList to update their internal state when the underlying array changes, rather than being replaced and losing state.

**Reuse pattern:**
```lua
function MyWrapper:new(variable)
    local existing = variable:getWrapper()
    if existing then
        existing.value = variable:getValue()  -- Update value reference
        -- Sync internal state with new value...
        return existing
    end

    -- Create new wrapper only if none exists
    local wrapper = {
        variable = variable,
        value = variable:getValue(),
        -- ...internal state...
    }
    setmetatable(wrapper, self)
    return wrapper
end
```

### Wrapper Lifecycle

1. Variable created with `wrapper=TypeName` in path properties
2. `Resolver.CreateWrapper(variable)` called
3. Wrapper constructor: `TypeName:new(variable)`
4. Wrapper registered in object registry (stands in for navigation)
5. On value changes: `Resolver.CreateWrapper(variable)` called again
   - Wrapper can return existing instance or create new one
6. On variable destroy: wrapper destroyed, managed objects cleaned up

### Built-in Wrappers

**ViewList** - Manages array items with selection support:
- Stores `variable` property (the Variable object)
- Accesses array via `variable:getValue()`
- Maintains `items` array of ViewListItem objects
- Maintains `selectionIndex` for frontend selection state
- On reuse: syncs ViewListItems with new array (see crc-ViewList.md)

### Custom Wrappers

Applications can define custom wrappers for specialized transformations:
- Computed display values
- Filtered/sorted views
- Aggregated data with selection state

### Wrapper Resolution (Auto-discovery)

1. Check global Go wrapper registry (auto-populated via `init()`)
2. Look up Lua global by name - if exists with proper structure, use it

No explicit registration calls required (frictionless development).

## Sequences

- seq-wrapper-transform.md: Wrapper creation and reuse
- seq-viewlist-presenter-sync.md: ViewList wrapper syncs ViewListItems
