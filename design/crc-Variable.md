# Variable

**Source Spec:** protocol.md

## Responsibilities

### Knows
- id: Unique integer identifier (1+, 0 = null reference)
- parentId: Optional parent variable ID for tree structure
- value: JSON value (string, number, boolean, null, array, or object reference)
- properties: Map of string properties (empty string = unset)
- watchCount: Number of observers watching this variable
- monitoredValue: Value used for change detection (shallow copy for arrays)
- storedValue: Computed value sent to frontend (raw or wrapped)
- wrapperInstance: Internal wrapper object (nil if no wrapper)

### Does
- getValue: Return the stored value (for frontend)
- setValue: Update raw value, detect changes, recompute stored value
- getProperty: Get a property by name
- setProperty: Set a property value
- isStandardVariable: Check if registered with @NAME id pattern
- isObjectReference: Check if value is {obj: ID} form
- isUnbound: Check if storage is in UI server (not external backend)
- hasWrapper: Check if wrapperInstance is set
- getMonitoredValue: Return shallow copy used for change detection
- detectChanges: Compare raw value to monitored value
- initWrapper: Create wrapper instance from wrapper property type name
- computeStoredValue: Call wrapper.computeValue(rawValue) or use raw value

## Collaborators

- VariableStore: Persists variable state
- WatchManager: Manages watch subscriptions
- ProtocolHandler: Receives create/update/destroy messages
- Wrapper: Stored internally, transforms raw value to stored value

## Notes

### Dual Value Architecture

Variables maintain two values:

1. **Monitored value** - Used for change detection
   - For arrays: a shallow copy to detect content changes
   - For other values: the raw value from path resolution

2. **Stored value** - The variable's actual value, sent to frontend
   - Without wrapper: same as raw value (in "value JSON" form)
   - With wrapper: result of `wrapper.computeValue(rawValue)`

### Wrapper Lifecycle

When a variable is created with `wrapper=TypeName` in path properties:
1. A wrapper object of that type is instantiated: `Constructor(variable)`
2. The wrapper instance is stored internally in the variable (not as a property)
3. The wrapper persists for the lifetime of the variable
4. When destroyed, the wrapper's cleanup is called

The wrapper can access path properties (like `item=ContactPresenter`) from the variable's properties.

Set via path property syntax: `contacts?wrapper=ViewList&item=ContactPresenter`

## Sequences

- seq-create-variable.md: Creating a new variable
- seq-update-variable.md: Updating variable value/properties
- seq-watch-variable.md: Subscribing to variable changes
- seq-destroy-variable.md: Destroying variable and children
- seq-wrapper-transform.md: Wrapper transforms outgoing value
