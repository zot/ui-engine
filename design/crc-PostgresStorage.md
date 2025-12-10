# PostgresStorage

**Source Spec:** deployment.md

## Responsibilities

### Knows
- pool: PostgreSQL connection pool
- tableName: Table for variable storage
- prepared: Prepared statements for common operations

### Does
- store: INSERT ON CONFLICT UPDATE variable row
- load: SELECT variable by ID
- delete: DELETE variable row
- loadChildren: SELECT WHERE parent_id = $1
- exists: SELECT EXISTS check
- clear: TRUNCATE table
- migrate: Create/update schema with migrations
- beginTransaction: BEGIN
- commit: COMMIT
- rollback: ROLLBACK
- acquireConnection: Get connection from pool
- releaseConnection: Return connection to pool

## Collaborators

- StorageBackend: Implements interface
- VariableStore: Primary consumer

## Sequences

- seq-store-variable.md: PostgreSQL storage path
- seq-retrieve-variable.md: PostgreSQL retrieval path
