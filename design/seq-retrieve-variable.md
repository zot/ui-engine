# Sequence: Retrieve Variable

**Source Spec:** deployment.md, data-models.md
**Use Case:** Loading variable from storage backend

## Participants

- VariableStore: Variable storage manager
- StorageBackend: Abstract storage interface
- MemoryStorage: In-memory storage
- SQLiteStorage: SQLite database storage

## Sequence

```
     VariableStore         StorageBackend         MemoryStorage         SQLiteStorage
        |                      |                      |                      |
        |---load(varId)------->|                      |                      |
        |                      |                      |                      |
        |                      |          [Memory storage path]              |
        |                      |---load()------------>|                      |
        |                      |                      |                      |
        |                      |                      |---map.get(id)------->|
        |                      |                      |                      |
        |                      |<--data---------------|                      |
        |                      |                      |                      |
        |                      |          [SQLite storage path]              |
        |                      |---load()-------------------------------------->|
        |                      |                      |                      |
        |                      |                      |                      |---SELECT WHERE id--->
        |                      |                      |                      |
        |                      |<--row data------------------------------------|
        |                      |                      |                      |
        |                      |---deserialize------->|                      |
        |                      |   (from JSON)        |                      |
        |                      |                      |                      |
        |                      |---createVariable---->|                      |
        |                      |   (from data)        |                      |
        |                      |                      |                      |
        |<--variable-----------|                      |                      |
        |                      |                      |                      |
```

## Notes

- Variable ID used as primary key
- Memory storage O(1) map lookup
- Database storage uses indexed query
- JSON deserialized to Variable object
- Null returned if variable not found
- loadChildren uses parent_id index for efficient child retrieval
