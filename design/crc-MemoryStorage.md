# MemoryStorage

**Source Spec:** deployment.md

## Responsibilities

### Knows
- variables: In-memory map of variable ID to variable data
- objects: In-memory map of object ID to object data
- childIndex: Map of parent ID to list of child IDs

### Does
- store: Add/update variable in memory
- load: Retrieve variable from memory
- delete: Remove variable and update child index
- loadChildren: Return children from child index
- exists: Check map key existence
- clear: Empty all maps

## Collaborators

- StorageBackend: Implements interface
- VariableStore: Primary consumer

## Sequences

- seq-store-variable.md: Memory storage path
- seq-retrieve-variable.md: Memory retrieval path
