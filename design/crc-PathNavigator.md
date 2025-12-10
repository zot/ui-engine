# PathNavigator

**Source Spec:** protocol.md, libraries.md

## Responsibilities

### Knows
- pathCache: Cache of resolved paths
- reflectionCache: Cache of type reflection data

### Does
- resolve: Navigate path to get value (e.g., "father.name")
- resolveForWrite: Navigate path and return parent + key for setting
- parsePath: Split path into segments
- navigateSegment: Handle single path segment (property, index, method)
- handleMethodCall: Execute method call segment (e.g., "getName()")
- handleArrayIndex: Navigate to array element (1-based)
- handleParentTraversal: Handle ".." segment
- resolveStandardVariable: Handle "@name" prefix

## Collaborators

- BackendConnection: Uses for path navigation
- BindingEngine: Frontend path resolution
- Variable: Target of path navigation
- VariableStore: Standard variable lookup

## Sequences

- seq-path-resolve.md: Full path resolution
- seq-bind-element.md: Frontend path binding
