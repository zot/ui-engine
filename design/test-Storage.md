# Test Design: Storage System

**Source Specs**: deployment.md, data-models.md
**CRC Cards**: crc-StorageBackend.md, crc-MemoryStorage.md, crc-SQLiteStorage.md, crc-PostgresStorage.md
**Sequences**: seq-store-variable.md, seq-retrieve-variable.md

## Overview

Tests for variable persistence across Memory, SQLite, and PostgreSQL backends.

## Test Cases

### Test: Memory storage store and load

**Purpose**: Verify basic memory storage

**Input**:
- store(variable) with id=5, value="test"
- load(5)

**References**:
- CRC: crc-MemoryStorage.md - "Does: store, load"
- Sequence: seq-store-variable.md

**Expected Results**:
- Variable stored in map
- load returns same variable
- Value "test" preserved

---

### Test: Memory storage loadChildren

**Purpose**: Verify child variable retrieval

**Input**:
- Store parent (id=1, parentId=0)
- Store child1 (id=2, parentId=1)
- Store child2 (id=3, parentId=1)
- loadChildren(1)

**References**:
- CRC: crc-MemoryStorage.md - "Does: loadChildren"

**Expected Results**:
- Returns [child1, child2]
- Order not guaranteed
- Does not include grandchildren

---

### Test: Memory storage delete

**Purpose**: Verify variable removal

**Input**:
- store(variable) with id=5
- delete(5)
- load(5)

**References**:
- CRC: crc-MemoryStorage.md - "Does: delete"

**Expected Results**:
- Variable removed from map
- load(5) returns null
- Child index updated

---

### Test: Memory storage clear

**Purpose**: Verify complete data wipe

**Input**:
- Store multiple variables
- clear()
- load any

**References**:
- CRC: crc-MemoryStorage.md - "Does: clear"

**Expected Results**:
- All variables removed
- All objects removed
- Child index empty

---

### Test: SQLite storage store and load

**Purpose**: Verify SQLite persistence

**Input**:
- store(variable) with id=5, value="test"
- load(5)

**References**:
- CRC: crc-SQLiteStorage.md - "Does: store, load"
- Sequence: seq-store-variable.md

**Expected Results**:
- Variable in database table
- load returns variable
- Value correctly deserialized

---

### Test: SQLite storage upsert

**Purpose**: Verify INSERT OR REPLACE behavior

**Input**:
- store(variable) with id=5, value="old"
- store(variable) with id=5, value="new"
- load(5)

**References**:
- CRC: crc-SQLiteStorage.md - "Does: store"

**Expected Results**:
- Only one row for id=5
- Value is "new"
- No duplicate key error

---

### Test: SQLite storage loadChildren with index

**Purpose**: Verify parent_id index usage

**Input**:
- Store 100 children under parent id=1
- loadChildren(1)

**References**:
- CRC: crc-SQLiteStorage.md - "Does: loadChildren"

**Expected Results**:
- All 100 children returned
- Query uses index
- Performance acceptable

---

### Test: SQLite storage transaction commit

**Purpose**: Verify atomic operations

**Input**:
- beginTransaction()
- store(var1), store(var2)
- commit()

**References**:
- CRC: crc-SQLiteStorage.md - "Does: beginTransaction, commit"

**Expected Results**:
- Both variables stored
- Visible after commit
- Atomic (all or nothing)

---

### Test: SQLite storage transaction rollback

**Purpose**: Verify rollback behavior

**Input**:
- beginTransaction()
- store(var1), store(var2)
- rollback()

**References**:
- CRC: crc-SQLiteStorage.md - "Does: rollback"

**Expected Results**:
- Neither variable stored
- Database unchanged
- No partial writes

---

### Test: SQLite storage migration

**Purpose**: Verify schema creation

**Input**:
- New database file
- migrate()

**References**:
- CRC: crc-SQLiteStorage.md - "Does: migrate"

**Expected Results**:
- Tables created
- Indexes created
- Ready for operations

---

### Test: PostgreSQL storage store and load

**Purpose**: Verify PostgreSQL persistence

**Input**:
- store(variable) with id=5, value="test"
- load(5)

**References**:
- CRC: crc-PostgresStorage.md - "Does: store, load"
- Sequence: seq-store-variable.md

**Expected Results**:
- Variable in database
- load returns variable
- JSON serialization correct

---

### Test: PostgreSQL storage connection pool

**Purpose**: Verify pool management

**Input**:
- Multiple concurrent operations

**References**:
- CRC: crc-PostgresStorage.md - "Does: acquireConnection, releaseConnection"

**Expected Results**:
- Connections reused
- Pool limits respected
- No connection leaks

---

### Test: PostgreSQL storage ON CONFLICT update

**Purpose**: Verify upsert behavior

**Input**:
- INSERT with existing id

**References**:
- CRC: crc-PostgresStorage.md - "Does: store"

**Expected Results**:
- Row updated, not duplicated
- No constraint violation
- Values overwritten

---

### Test: StorageBackend interface polymorphism

**Purpose**: Verify interchangeable backends

**Input**:
- Same operations on Memory, SQLite, PostgreSQL

**References**:
- CRC: crc-StorageBackend.md

**Expected Results**:
- Identical behavior
- Same results
- Interface contract maintained

---

### Test: Storage survives restart (SQLite)

**Purpose**: Verify persistence across restarts

**Input**:
- Store variable
- Restart server
- Load variable

**References**:
- CRC: crc-SQLiteStorage.md

**Expected Results**:
- Variable still exists
- Value unchanged
- No data loss

---

### Test: Memory storage lost on restart

**Purpose**: Verify ephemeral nature of memory storage

**Input**:
- Store variable in memory
- Server restart simulation

**References**:
- CRC: crc-MemoryStorage.md

**Expected Results**:
- Variables gone after restart
- Expected behavior for memory
- Use case: ephemeral UIs

---

## Coverage Summary

**Responsibilities Covered:**
- StorageBackend: store, load, delete, loadChildren, exists, clear, beginTransaction, commit, rollback
- MemoryStorage: store, load, delete, loadChildren, exists, clear (map operations)
- SQLiteStorage: store, load, delete, loadChildren, exists, clear, migrate, beginTransaction, commit, rollback
- PostgresStorage: store, load, delete, loadChildren, exists, clear, migrate, acquireConnection, releaseConnection

**Scenarios Covered:**
- seq-store-variable.md: All paths (Memory, SQLite, PostgreSQL)
- seq-retrieve-variable.md: All paths

**Gaps**: None identified
