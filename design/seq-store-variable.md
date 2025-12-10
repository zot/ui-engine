# Sequence: Store Variable

**Source Spec:** deployment.md, data-models.md
**Use Case:** Persisting variable to storage backend

## Participants

- VariableStore: Variable storage manager
- StorageBackend: Abstract storage interface
- MemoryStorage: In-memory storage
- SQLiteStorage: SQLite database storage

## Sequence

```
     VariableStore         StorageBackend         MemoryStorage         SQLiteStorage
        |                      |                      |                      |
        |---store(variable)--->|                      |                      |
        |                      |                      |                      |
        |                      |---serialize--------->|                      |
        |                      |   (to JSON)          |                      |
        |                      |                      |                      |
        |                      |          [Memory storage path]              |
        |                      |---store()----------->|                      |
        |                      |                      |                      |
        |                      |                      |---map.set(id,data)-->|
        |                      |                      |                      |
        |                      |                      |---updateChildIndex-->|
        |                      |                      |                      |
        |                      |<--success------------|                      |
        |                      |                      |                      |
        |                      |          [SQLite storage path]              |
        |                      |---store()------------------------------------>|
        |                      |                      |                      |
        |                      |                      |                      |---INSERT/UPDATE--->
        |                      |                      |                      |   (variables table)
        |                      |                      |                      |
        |                      |<--success------------------------------------|
        |                      |                      |                      |
        |<--success------------|                      |                      |
        |                      |                      |                      |
```

## Notes

- Variable serialized to JSON for storage
- Memory storage uses in-memory maps
- SQLite uses INSERT ON CONFLICT UPDATE
- PostgreSQL similar to SQLite with connection pooling
- Parent ID indexed for efficient child queries
- Transactions used for atomic operations
