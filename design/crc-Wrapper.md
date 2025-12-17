# Wrapper

**Source Spec:** protocol.md, libraries.md

## Responsibilities

### Knows
- variable: The Variable object (received in constructor, stored for later access)
- value: The variable's value (from `variable:getValue()`, stored for convenience)
- managedObjects: Objects created and managed by this wrapper (e.g., ViewListItems)

### Does
- new(variable): Constructor receives Variable object, returns new or existing wrapper
- sync: Update internal state when value changes (on wrapper reuse)
- destroy (optional): Clean up all managed objects when variable destroyed

## Collaborators

- Variable: Stores wrapper instance internally, provides getValue() and getWrapper()
- WrapperManager: Calls the appropriate factory to create a wrapper instance.
- ObjectRegistry: Registers wrapper object for child path navigation
- LuaRuntime: Hosts wrapper implementation (for embedded Lua)

## Notes

### Wrapper Behavior

The wrapper object itself **stands in for the variable's value** when child variables navigate paths. The wrapper is registered in the object registry and becomes the variable's navigation value. There is no `computeValue()` method - the wrapper IS the value.

### Optional `Wrapper` Interface

There is no formal `Wrapper` interface that all wrappers must implement. However, a wrapper can optionally implement a `Destroy() error` method. If it does, the method will be called when the variable is destroyed.

### Wrapper Creation and Reuse

`WrapperManager.CreateWrapper(variable)` is called **whenever the variable's value changes**. The factory can:

1. **Return existing wrapper** - Preserves internal state (selection, scroll position)
2. **Return new wrapper** - Creates fresh state
3. **Return nil** - No wrapper needed

This enables stateful wrappers like ViewList to update their internal state when the underlying array changes, rather than being replaced and losing state.

**Reuse pattern:**
```go
func NewMyWrapper(runtime *Runtime, variable WrapperVariable) interface{} {
    // variable.getWrapper() is not a method on the variable, it's an internal lookup.
    // The factory needs a way to check if a wrapper already exists for the variable.
    // For now, we assume the factory is only called once.
    
    wrapper := &MyWrapper{
        variable: variable,
        value:    variable.GetValue(),
        // ...internal state...
    }
    // sync logic here
    return wrapper
}
```

### Wrapper Lifecycle

1. Variable created with `wrapper=TypeName` in path properties
2. `WrapperManager.CreateWrapper(variable)` called
3. Wrapper factory for `TypeName` is called: `NewTypeName(runtime, variable)`
4. Wrapper object is stored in the variable's `wrapperInstance` field.
5. On value changes: The wrapper's sync logic is triggered (e.g., in the constructor).
6. On variable destroy: If the wrapper has a `Destroy()` method, it is called.

### Factory Registries

There are two types of factory registries for creating objects:

1.  **Create Factory:** Used for the `create` property. It creates a new object from a value.
2.  **Wrapper Factory:** Used for the `wrapper` property. It creates a new wrapper instance from a variable.

Both registries are populated automatically by `init()` functions in the Go code.
