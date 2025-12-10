# StorageBackend

**Source Spec:** deployment.md, data-models.md

## Responsibilities

### Knows
- type: Storage type (Memory, SQLite, PostgreSQL)
- connectionString: Database connection info (if applicable)

### Does
- store: Persist variable to storage
- load: Retrieve variable from storage
- delete: Remove variable from storage
- loadChildren: Get all child variables of parent
- exists: Check if variable exists
- clear: Remove all data (for testing/reset)
- beginTransaction: Start atomic operation
- commit: Complete atomic operation
- rollback: Cancel atomic operation

## Collaborators

- VariableStore: Uses for persistence
- MemoryStorage: In-memory implementation
- SQLiteStorage: SQLite implementation
- PostgresStorage: PostgreSQL implementation

## Sequences

- seq-store-variable.md: Variable persistence
- seq-retrieve-variable.md: Variable loading
