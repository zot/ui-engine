# SQLiteStorage

**Source Spec:** deployment.md

## Responsibilities

### Knows
- db: SQLite database connection
- tableName: Table for variable storage
- prepared: Prepared statements for common operations

### Does
- store: INSERT or UPDATE variable row
- load: SELECT variable by ID
- delete: DELETE variable row
- loadChildren: SELECT WHERE parentId = ?
- exists: SELECT EXISTS check
- clear: DROP and recreate table
- migrate: Create/update schema
- beginTransaction: START TRANSACTION
- commit: COMMIT
- rollback: ROLLBACK

## Collaborators

- StorageBackend: Implements interface
- VariableStore: Primary consumer

## Sequences

- seq-store-variable.md: SQLite storage path
- seq-retrieve-variable.md: SQLite retrieval path
