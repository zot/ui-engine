# PathNavigator

**Source Spec:** protocol.md, libraries.md, viewdefs.md

## Responsibilities

### Knows
- pathCache: Cache of resolved paths
- reflectionCache: Cache of type reflection data

### Does
- resolve: Navigate path to get value (e.g., "father.name"), returns null/undefined if any segment is nullish
- resolveForWrite: Navigate path and return parent + key for setting, returns null if any intermediate segment is nullish
- resolveForMethod: Navigate path to get target object and method name for ui-action dispatch
- parsePath: Split path into segments
- navigateSegment: Handle single path segment (property, index, method), returns null/undefined for nullish values
- handleMethodCall: Execute method call segment (e.g., "getName()")
- handleMethodCallWithArg: Handle method call with _ argument (e.g., "method(_)")
- handleArrayIndex: Navigate to array element (1-based)
- handleParentTraversal: Handle ".." segment
- resolveStandardVariable: Handle "@name" prefix
- isNullish: Check if value is null or undefined

## Nullish Path Handling

Path traversal uses nullish coalescing behavior (like JavaScript's `?.` operator):
- **Read direction:** If any segment resolves to null/undefined, returns null/undefined (no error)
- **Write direction:** If any intermediate segment is nullish, resolveForWrite returns null (caller sends `error` message with code `path-failure`, allowing UI to show error indicator)

## ui-action Path Dispatch

For ui-action bindings, paths end in method calls:
- `presenter.save()` - Navigate to `presenter`, call `save()` with no args
- `delegate.run(_)` - Navigate to `delegate`, call `run(value)` with update message value

## Collaborators

- LuaSession: Uses for path-based action dispatch
- BindingEngine: Frontend path resolution
- Variable: Target of path navigation
- VariableStore: Standard variable lookup

## Sequences

- seq-path-resolve.md: Full path resolution
- seq-bind-element.md: Frontend path binding
- seq-lua-handle-action.md: Path-based action dispatch
