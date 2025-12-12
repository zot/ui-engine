# Wrapper

**Source Spec:** protocol.md

## Responsibilities

### Knows
- variable: The variable being wrapped (received in constructor)
- managedObjects: Objects created and managed by this wrapper (e.g., ViewItems, presenters)

### Does
- computeValue: Transform raw value into stored value for frontend
- destroy: Clean up all managed objects when variable destroyed
- createManagedObject: Create objects (e.g., presenters) tied to wrapper lifecycle
- destroyManagedObject: Clean up managed objects

## Collaborators

- Variable: Stores wrapper instance internally, calls computeValue on changes
- LuaRuntime: Hosts wrapper implementation (for embedded Lua)
- VariableStore: Stores managed objects created by wrapper

## Notes

### Wrapper Interface

Wrappers implement a simple interface:

```
Wrapper {
    Constructor(variable)        -- Receives variable, stores reference
    computeValue(rawValue) -> storedValue  -- Called when monitored value changes
    destroy()                    -- Called when variable is destroyed
}
```

The wrapper constructor receives the variable, allowing it to:
- Access variable properties (e.g., `item=ContactPresenter`)
- Store a reference for later use
- Set up initial state

### Wrapper Lifecycle

1. Variable created with `wrapper=TypeName` in properties
2. Wrapper instantiated: `Constructor(variable)`
3. Wrapper stored internally in variable (not as a property)
4. On value changes: `wrapper.computeValue(rawValue)` returns stored value
5. On variable destroy: `wrapper.destroy()` cleans up

### Built-in Wrappers

**ViewList** - Transforms array of domain refs to array of ViewItem refs:
- Constructor receives variable, reads `item` property for presenter type
- computeValue creates/syncs ViewItem objects for each domain item
- Each ViewItem has: baseItem (domain ref), item (same or wrapped presenter), list, index
- Maintains parallel ViewItem array
- Returns ViewItem refs as stored value

### Custom Wrappers

Applications can define custom wrappers for specialized transformations:
- Computed display values
- Filtered/sorted views
- Aggregated data

### Wrapper Registration

Wrappers are registered by type name:

```lua
ui.registerWrapper("ViewList", ViewListWrapper)
ui.registerWrapper("CountDisplay", CountDisplayWrapper)
```

## Sequences

- seq-wrapper-transform.md: Wrapper transforms outgoing value
- seq-viewlist-presenter-sync.md: ViewList wrapper syncs presenters
