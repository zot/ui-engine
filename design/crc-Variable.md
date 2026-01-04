# Variable (Frontend)

**Source Spec:** protocol.md

**Note:** Backend variables are managed by the change-tracker library (separate project). This CRC describes the frontend Variable representation only.

## Responsibilities

### Knows
- varId: Unique integer identifier (1+, 0 = null reference)
- parentId: Optional parent variable ID
- value: The variable's value (unknown type - could be primitive, object reference, etc.)
- properties: Map of string properties (e.g., path, type, namespace, access)
- widget: Optional reference to the Widget that created this variable (direct pointer)
- unbound: Optional flag indicating variable was destroyed

### Does
- (Data structure only - no methods, managed by VariableStore)

## Collaborators

- VariableStore: Caches variables, handles updates from backend
- BindingEngine: Creates child variables for bindings, watches for changes

## Notes

### Frontend vs Backend

The frontend Variable is a simple cache of what the backend sends:
- Receives updates via WebSocket messages
- Stores value and properties locally for UI binding
- No change detection or path resolution (backend handles that)

### Object References

Variable values may be object references (`{obj: number}`) rather than actual data. The frontend renders these using viewdefs - it never has access to the actual object contents.

### Properties

Common properties (set by backend or frontend):
- `path`: Navigation path from parent (set by frontend bindings)
- `type`: Object type for viewdef lookup (set by backend)
- `namespace`: Viewdef namespace override
- `fallbackNamespace`: Secondary namespace for viewdef lookup
- `access`: Access mode (r, w, rw, action)

## Sequences

- seq-create-variable.md: Frontend requests variable creation
- seq-update-variable.md: Backend sends value/property updates
- seq-destroy-variable.md: Variable cleanup
