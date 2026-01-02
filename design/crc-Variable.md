# Variable

**Source Spec:** protocol.md, viewdefs.md

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
- elementId: HTML ID of the element that created this variable (frontend only, vended if element has no ID)
- namespace: Viewdef namespace (from closest `ui-namespace` element, or inherited from parent)
- fallbackNamespace: Fallback namespace for viewdef lookup (inherited from parent, set by wrappers like ViewList)

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
- resolveNamespace: Determine namespace from closest ui-namespace element or parent variable
- inheritFallbackNamespace: Copy `fallbackNamespace` from parent variable if not explicitly set

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

### Element ID Tracking

Each variable tracks the **element ID** of the element that created it (frontend only):
- If the element has an ID, use it
- If the element doesn't have an ID, vend one and assign it to the element

Element ID tracking is used for:
- Namespace inheritance (finding closest `ui-namespace` element)
- Debugging and inspection
- Understanding the variable-element relationship

### Namespace Resolution

When creating a view's variable, namespace is determined by:

1. Find the closest element with `ui-namespace` using `element.closest('[ui-namespace]')`
2. If found and either:
   - There's no parent variable, OR
   - The parent variable's element contains the found element

   Then use that namespace value.
3. Otherwise, inherit `namespace` property from the parent variable (if set)

This allows intermediate elements with `ui-namespace` to override the parent variable's namespace within a viewdef.

### Fallback Namespace Properties

Variables have `fallbackNamespace` for secondary viewdef lookup:
- Always inherited from parent variable
- Set by backend wrappers (e.g., ViewList sets `fallbackNamespace: "list-item"`)
- Used when `namespace` viewdef doesn't exist

## Sequences

- seq-create-variable.md: Creating a new variable
- seq-update-variable.md: Updating variable value/properties
- seq-watch-variable.md: Subscribing to variable changes
- seq-destroy-variable.md: Destroying variable and children
- seq-wrapper-transform.md: Wrapper creation and storage
