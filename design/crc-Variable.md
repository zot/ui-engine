# Variable

**Source Spec:** protocol.md

## Responsibilities

### Knows
- id: Unique integer identifier (1+, 0 = null reference)
- parentId: Optional parent variable ID for tree structure
- value: The raw value of the variable (an `interface{}`)
- properties: Map of string properties (empty string = unset)
- watchCount: Number of observers watching this variable
- monitoredValue: Value used for change detection (`interface{}`)
- storedValue: The value sent to the frontend; this is the `wrapperInstance` if it exists, otherwise it is the raw `value`.
- wrapperInstance: Internal wrapper object (`interface{}`), nil if no wrapper

### Does
- getValue: Return the raw value
- setValue: Update raw value, detect changes
- getProperty: Get a property by name
- setProperty: Set a property value
- isStandardVariable: Check if registered with @NAME id pattern
- hasWrapper: Check if wrapperInstance is set
- getMonitoredValue: Return shallow copy used for change detection
- detectChanges: Compare raw value to monitored value
- initWrapper: Create wrapper instance from wrapper property type name using the `WrapperFactory`.

## Collaborators

- VariableStore: Persists variable state
- WatchManager: Manages watch subscriptions
- ProtocolHandler: Receives create/update/destroy messages
- Wrapper: Stored internally, acts as a stand-in for the variable's value.

## Notes

### Dual Value Architecture

Variables maintain two values:

1. **Monitored value** - Used for change detection. For arrays, this is a shallow copy to detect content changes. For other values, it's the raw value from path resolution.

2. **Stored value** - The variable's actual value, sent to the frontend.
   - Without wrapper: same as raw value (in "value JSON" form).
   - With wrapper: the `wrapperInstance` itself.

### Wrapper Lifecycle

When a variable is created with `wrapper=TypeName` in path properties:
1. A wrapper object of that type is instantiated via the `WrapperFactory`: `NewTypeName(runtime, variable)`
2. The wrapper instance is stored internally in the variable's `wrapperInstance` field.
3. The `storedValue` is set to the `wrapperInstance`.
4. The wrapper persists for the lifetime of the variable.
5. When destroyed, if the wrapper has a `Destroy()` method, it is called.

The wrapper can access path properties (like `item=ContactPresenter`) from the variable's properties.

Set via path property syntax: `contacts?wrapper=ViewList&item=ContactPresenter`

## Sequences

- seq-create-variable.md: Creating a new variable
- seq-update-variable.md: Updating variable value/properties
- seq-watch-variable.md: Subscribing to variable changes
- seq-destroy-variable.md: Destroying variable and children
- seq-wrapper-transform.md: Wrapper creation and storage
