# ObjectRegistry

**Source Spec:** libraries.md

## Responsibilities

### Knows
- weakRefs: Map of weak.Pointer to variable ID (weak reference to object -> varID)
- strongRefs: Map of variable ID to object pointer (varID -> strong reference for registration)
- cleanupInterval: Interval for automatic cleanup of GC'd entries
- mu: Mutex for thread-safe access

### Does
- register: Associate object pointer with variable ID using weak reference
- unregister: Remove object from registry when variable is destroyed
- lookup: Find variable ID for an object during serialization (returns nil if not found or GC'd)
- serializeWithRefs: Custom JSON marshaler that emits {"obj": id} for registered objects
- cleanup: Scan for and remove entries where weak reference has been collected
- startCleanup: Begin automatic periodic cleanup goroutine
- stopCleanup: Stop cleanup goroutine

## Collaborators

- ChangeDetector: Uses registry during serialization to emit object references
- PathNavigator: Resolves paths to objects which may be in registry
- BackendConnection: Triggers registration when paths are watched

## Sequences

- seq-object-registry.md: Path watch, serialization, and cleanup cycle

## Notes

- **Go 1.25+ required**: Uses `weak` package for weak pointers
- **Identity-based**: Same object in multiple locations serializes to same `{"obj": id}`
- **Frictionless**: Domain objects require no modification - no interfaces, embedded IDs, or registration methods
- **Automatic cleanup**: Objects can be GC'd when no longer referenced by application code
- **Thread-safe**: All operations protected by mutex
- Key insight: Frontend creates variables by watching paths (from viewdefs). Backend just has domain objects it modifies directly. ObjectRegistry bridges these by mapping object identity to variable ID.

## Implementation Considerations

- Use `weak.Make[T](ptr)` to create weak pointers
- Use `wp.Value()` to get strong pointer (returns nil if collected)
- Weak pointer can be used as map key via `weak.Pointer[T]`
- Consider using `runtime.SetFinalizer` as alternative for pre-1.25 compatibility
