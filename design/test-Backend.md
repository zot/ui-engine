# Test Design: Backend Library

**Source Specs**: libraries.md
**CRC Cards**: crc-BackendConnection.md, crc-PathNavigator.md
**Sequences**: seq-backend-connect.md, seq-backend-refresh.md, seq-path-resolve.md

**External Package**: Change detection and object registry are provided by `change-tracker` (`github.com/zot/change-tracker`) which has its own test suite.

## Overview

Tests for Go/Lua backend library supporting connection and path navigation. Change detection and object registry functionality is tested by the change-tracker package.

## Test Cases

### Test: Backend connection establishment

**Purpose**: Verify connection to UI server

**Input**:
- connect(url, rootValue)

**References**:
- CRC: crc-BackendConnection.md - "Does: connect"
- Sequence: seq-backend-connect.md

**Expected Results**:
- WebSocket opened
- Session bound
- Root value sent to variable 1

---

### Test: Backend disconnect with cleanup hook

**Purpose**: Verify disconnect callback

**Input**:
- onClose(callback) registered
- Connection closed

**References**:
- CRC: crc-BackendConnection.md - "Does: disconnect, onClose"

**Expected Results**:
- Callback invoked
- Resources cleaned up
- Reconnection possible

---

### Test: Backend send message

**Purpose**: Verify message sending

**Input**:
- send(update message)

**References**:
- CRC: crc-BackendConnection.md - "Does: send"

**Expected Results**:
- Message sent via WebSocket
- Proper serialization
- Delivery confirmed

---

### Test: Backend receive message

**Purpose**: Verify message receipt handling

**Input**:
- Server sends watch message

**References**:
- CRC: crc-BackendConnection.md - "Does: receive"

**Expected Results**:
- Message parsed
- Handler invoked
- Response generated

---

### Test: Path resolve simple property

**Purpose**: Verify property navigation

**Input**:
- Object: {name: "John"}
- resolve("name")

**References**:
- CRC: crc-PathNavigator.md - "Does: resolve"
- Sequence: seq-path-resolve.md

**Expected Results**:
- Returns "John"
- Correct type
- No side effects

---

### Test: Path resolve nested property

**Purpose**: Verify dot notation navigation

**Input**:
- Object: {father: {name: "Bob"}}
- resolve("father.name")

**References**:
- CRC: crc-PathNavigator.md - "Does: navigateSegment"

**Expected Results**:
- Returns "Bob"
- Intermediate object traversed
- Null-safe handling

---

### Test: Path resolve array index

**Purpose**: Verify 1-based array indexing

**Input**:
- Object: {items: ["a", "b", "c"]}
- resolve("items.2")

**References**:
- CRC: crc-PathNavigator.md - "Does: handleArrayIndex"

**Expected Results**:
- Returns "b" (index 2 = second element)
- 1-based indexing
- Bounds checking

---

### Test: Path resolve method call

**Purpose**: Verify method invocation in path

**Input**:
- Object with getName() method returning "Alice"
- resolve("getName()")

**References**:
- CRC: crc-PathNavigator.md - "Does: handleMethodCall"

**Expected Results**:
- Method called
- Returns "Alice"
- No arguments passed

---

### Test: Path resolve parent traversal

**Purpose**: Verify ".." segment handling

**Input**:
- Child object in parent
- resolve("..") from child

**References**:
- CRC: crc-PathNavigator.md - "Does: handleParentTraversal"

**Expected Results**:
- Returns parent object
- Maintains tree structure
- Works multiple levels

---

### Test: Path resolve standard variable

**Purpose**: Verify @name prefix handling

**Input**:
- @app registered with app object
- resolve("@app.url")

**References**:
- CRC: crc-PathNavigator.md - "Does: resolveStandardVariable"

**Expected Results**:
- @app resolved to registered variable
- .url navigated from there
- Returns current URL

---

### Test: Path resolve for write

**Purpose**: Verify parent + key extraction

**Input**:
- resolveForWrite("father.name")

**References**:
- CRC: crc-PathNavigator.md - "Does: resolveForWrite"

**Expected Results**:
- Returns {parent: father object, key: "name"}
- Enables setting value
- Works for all path types

---

## External Package Tests

The following functionality is tested by the `change-tracker` package (`github.com/zot/change-tracker`):

- **Change Detection**: addWatch, removeWatch, refresh, detectChange, scheduleRefresh
- **Object Registry**: register, unregister, lookup, serializeWithRefs, cleanup, weak references

See the change-tracker repository for its test suite.

---

## Coverage Summary

**Responsibilities Covered (UI-specific):**
- BackendConnection: connect, disconnect, send, receive, setRootValue, onClose, reconnect
- PathNavigator: resolve, resolveForWrite, parsePath, navigateSegment, handleMethodCall, handleArrayIndex, handleParentTraversal, resolveStandardVariable

**External Package (change-tracker):**
- ChangeDetector: addWatch, removeWatch, refresh, detectChange, scheduleRefresh, sendUpdates
- ObjectRegistry: register, unregister, lookup, serializeWithRefs, cleanup

**Scenarios Covered:**
- seq-backend-connect.md: All paths
- seq-backend-refresh.md: All paths
- seq-path-resolve.md: All paths

**Gaps**: None identified
