# Sequence: Wrapper Transform

**Source Spec:** protocol.md, libraries.md
**Use Case:** Variable with wrapper - wrapper creation and reuse on value changes

## Participants

- Variable: Has wrapper stored internally
- Resolver: Calls CreateWrapper on value changes, handles wrapper reuse
- Wrapper: Stands in for variable's value for path navigation
- ObjectRegistry: Registers wrapper for child path navigation
- LuaBackend: Notifies watchers of changes (per-session)

## Sequence

```
     Variable              Resolver               Wrapper            ObjectRegistry          LuaBackend
        |                      |                      |                      |                      |
        |   [on create with wrapper property]        |                      |                      |
        |                      |                      |                      |                      |
        |---CreateWrapper----->|                      |                      |                      |
        |   (variable)         |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---new(variable)----->|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getWrapper()------>|                      |
        |                      |                      |   (returns nil)      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getValue()-------->|                      |
        |                      |                      |                      |                      |
        |                      |                      |---getProperty------->|                      |
        |                      |                      |   ("item" type)      |                      |
        |                      |                      |                      |                      |
        |                      |<--wrapper------------|                      |                      |
        |                      |   (new instance)     |                      |                      |
        |                      |                      |                      |                      |
        |                      |---register-----------|--------------------->|                      |
        |                      |   (wrapper as        |                      |                      |
        |                      |    navigation value) |                      |                      |
        |                      |                      |                      |                      |
        |---storeWrapper------>|                      |                      |                      |
        |   (internal field)   |                      |                      |                      |
        |                      |                      |                      |                      |
        |   [on value change - wrapper reuse]        |                      |                      |
        |                      |                      |                      |                      |
        |---CreateWrapper----->|                      |                      |                      |
        |   (variable)         |                      |                      |                      |
        |                      |                      |                      |                      |
        |                      |---new(variable)----->|                      |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getWrapper()------>|                      |
        |                      |                      |   (returns existing) |                      |
        |                      |                      |                      |                      |
        |                      |                      |---getValue()-------->|                      |
        |                      |                      |   (get new value)    |                      |
        |                      |                      |                      |                      |
        |                      |                      |---sync()------------>|                      |
        |                      |                      |   (update internal   |                      |
        |                      |                      |    state)            |                      |
        |                      |                      |                      |                      |
        |                      |<--wrapper------------|                      |                      |
        |                      |   (same instance)    |                      |                      |
        |                      |                      |                      |                      |
        |                      |   [wrapper unchanged, no re-registration needed]                   |
        |                      |                      |                      |                      |
        |---notifyWatchers-----|--------------------------------------------->|                      |
        |   (wrapper is value) |                      |                      |                      |
        |                      |                      |                      |                      |
```

## Notes

### Wrapper as Navigation Value

- The wrapper object itself **stands in** for the variable's value when child variables navigate paths
- The wrapper is registered in the object registry
- There is no `computeValue()` method - the wrapper IS the value

### CreateWrapper Called on Value Changes

`Resolver.CreateWrapper(variable)` is called whenever the variable's value changes. The wrapper can:
- Return the **existing wrapper** to preserve internal state (selection, scroll position)
- Return a **new wrapper** if none exists yet
- Return `nil` if no wrapper is needed

### Wrapper Reuse Pattern

```lua
function Wrapper:new(variable)
    local existing = variable:getWrapper()
    if existing then
        existing.value = variable:getValue()  -- Update value reference
        existing:sync()                       -- Sync internal state
        return existing                       -- Return same instance
    end
    -- Create new wrapper only if none exists
    ...
end
```

### ViewList Example

For ViewList:
- Sets `fallbackNamespace: "list-item"` on the variable during creation
- On reuse, preserves:
  - `selectionIndex` - current selection state
  - `items` array - ViewListItems are synced, not recreated

See seq-viewlist-presenter-sync.md for ViewList-specific sync behavior.

### Wrapper Namespace Properties

Wrappers can set namespace-related properties on their variable:

```lua
function ViewList:new(variable)
    local existing = variable:getWrapper()
    if existing then
        -- Reuse existing wrapper...
        return existing
    end

    -- Set fallbackNamespace for 3-tier resolution
    variable:setProperty("fallbackNamespace", "list-item")

    -- Create new wrapper...
end
```

This enables the 3-tier namespace resolution: namespace -> fallbackNamespace -> DEFAULT.
