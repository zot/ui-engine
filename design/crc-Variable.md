# Variable

**Source Spec:** protocol.md

## Responsibilities

### Knows
- id: Unique integer identifier (1+, 0 = null reference)
- parentId: Optional parent variable ID for tree structure
- value: JSON value (string, number, boolean, null, array, or object reference)
- properties: Map of string properties (empty string = unset)
- watchCount: Number of observers watching this variable

### Does
- getValue: Return the current value
- setValue: Update the value and notify watchers
- getProperty: Get a property by name
- setProperty: Set a property value
- isStandardVariable: Check if registered with @NAME id pattern
- isObjectReference: Check if value is {obj: ID} form
- isUnbound: Check if storage is in UI server (not external backend)

## Collaborators

- VariableStore: Persists variable state
- WatchManager: Manages watch subscriptions
- ProtocolHandler: Receives create/update/destroy messages

## Sequences

- seq-create-variable.md: Creating a new variable
- seq-update-variable.md: Updating variable value/properties
- seq-watch-variable.md: Subscribing to variable changes
- seq-destroy-variable.md: Destroying variable and children
