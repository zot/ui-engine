# VariableStore

**Source Spec:** protocol.md, data-models.md

## Responsibilities

### Knows
- variables: Map of variable ID to Variable
- standardVariables: Map of @NAME to variable ID
- nextId: Counter for generating unique IDs
- storageBackend: Reference to persistence layer

### Does
- create: Create new variable with optional parent, value, properties
- get: Retrieve variable by ID
- getByName: Retrieve standard variable by @NAME
- update: Update variable value and/or properties
- destroy: Remove variable and all children recursively
- registerStandardVariable: Associate @NAME with variable ID
- getChildren: Find all variables with given parentId
- resolveObjectReference: Get object data for {obj: ID} references

## Collaborators

- Variable: Individual variable instances
- StorageBackend: Persistence layer (Memory/SQLite/PostgreSQL)
- ProtocolHandler: Receives protocol messages
- WatchManager: Notifies when variables change
- Config: Queries verbosity for variable operation logging (level 3)

## Sequences

- seq-create-variable.md: Variable creation flow
- seq-destroy-variable.md: Recursive destruction
- seq-store-variable.md: Persisting to storage
- seq-retrieve-variable.md: Loading from storage
